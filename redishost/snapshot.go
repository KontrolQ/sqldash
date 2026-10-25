package redishost

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"sqldash/database"
	"sqldash/models"
)

const (
	MaximumSnapshotBytes   int64 = 1024 * 1024 * 1024
	MaximumTransferTimeout       = 10 * time.Minute
)

func cloneLockPath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".clone.lock")
}

func validCloneFile(name string) bool {
	return len(name) == 74 && name[:6] == "clone-" && name[70:] == ".rdb" && validIdentity(name[6:70])
}

func visitCloneFiles(root *os.Root, visit func(string) error) error {
	directory, err := root.Open(".")
	if err != nil {
		return UnsafePath
	}
	defer directory.Close()
	var count int
	for {
		entries, readError := directory.ReadDir(64)
		for _, entry := range entries {
			count++
			if count > 4096 {
				return CapacityExceeded
			}
			if validCloneFile(entry.Name()) {
				if err := visit(entry.Name()); err != nil {
					return err
				}
			}
		}
		if errors.Is(readError, io.EOF) {
			return nil
		}
		if readError != nil {
			return UnsafePath
		}
	}
}

func removeCloneFile(root *os.Root, name string) error {
	information, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, 0) || information.Mode().Perm()&0o077 != 0 || information.Size() > MaximumSnapshotBytes {
		return UnsafePath
	}
	return root.Remove(name)
}

func cachedCloneSnapshot(ctx context.Context, source, target *models.Database) (*os.File, error) {
	stored, err := transferRecord(ctx, source, models.DatabaseReady)
	if err != nil {
		return nil, err
	}
	root, err := openSnapshotRoot(configurationDirectory(target.ID), 0)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	origin := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%d:%s", source.ID, source.ClusterID, target.ID, target.ClusterID)))
	name := fmt.Sprintf("clone-%x.rdb", origin)
	if err := visitCloneFiles(root, func(existing string) error {
		if existing != name {
			return IdentityMismatch
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if file, err := openSnapshotFile(root, name, 0); err == nil {
		return file, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := removeCloneFile(root, "clone.pending"); err != nil {
		return nil, err
	}
	pending, err := root.OpenFile("clone.pending", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, Unavailable
	}
	defer pending.Close()
	defer root.Remove("clone.pending")
	if err := Snapshot(ctx, stored, pending); err != nil {
		return nil, err
	}
	if err := pending.Sync(); err != nil {
		return nil, Unavailable
	}
	if err := pending.Close(); err != nil {
		return nil, Unavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := root.Rename("clone.pending", name); err != nil {
		return nil, Unavailable
	}
	if err := syncRestoreDirectory(configurationDirectory(target.ID)); err != nil {
		return nil, err
	}
	return openSnapshotFile(root, name, 0)
}

func CleanupClone(ctx context.Context, target *models.Database) error {
	if target == nil {
		return IdentityMismatch
	}
	candidate := *target
	candidate.State = models.DatabaseReady
	stored, err := transferRecord(ctx, &candidate, models.DatabaseReady)
	if err != nil {
		return err
	}
	instance, err := Instance(ctx, stored)
	if err != nil {
		return err
	}
	if !instance.Published {
		return PolicyUnconfirmed
	}
	lock, err := lockOwnedFile(cloneLockPath(target.ID), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer lock.Close()
	root, err := openSnapshotRoot(configurationDirectory(target.ID), 0)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := visitCloneFiles(root, func(name string) error { return removeCloneFile(root, name) }); err != nil {
		return err
	}
	if err := removeCloneFile(root, "clone.pending"); err != nil {
		return err
	}
	return syncRestoreDirectory(configurationDirectory(target.ID))
}

func transferLockPath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".transfer.lock")
}

func openSnapshotRoot(path string, owner int) (*os.Root, error) {
	if err := rejectSymlinks(path); err != nil {
		return nil, err
	}
	information, err := os.Lstat(path)
	forbidden := os.FileMode(0o077)
	if owner == 0 {
		forbidden = 0o022
	}
	if err != nil || !information.IsDir() || !verifyOwner(information, owner) || information.Mode().Perm()&forbidden != 0 {
		return nil, UnsafePath
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, UnsafePath
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(information, opened) || rejectSymlinks(path) != nil {
		root.Close()
		return nil, UnsafePath
	}
	return root, nil
}

func openSnapshotFile(root *os.Root, name string, owner int) (*os.File, error) {
	information, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !information.Mode().IsRegular() || !verifyOwner(information, owner) || information.Size() < 9 || information.Size() > MaximumSnapshotBytes || owner == 0 && information.Mode().Perm()&0o077 != 0 {
		return nil, UnsafePath
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, UnsafePath
	}
	opened, err := file.Stat()
	confirmed, confirmationError := root.Lstat(name)
	if err != nil || confirmationError != nil || !confirmed.Mode().IsRegular() || !os.SameFile(information, opened) || !os.SameFile(opened, confirmed) || !verifyOwner(opened, owner) || owner == 0 && opened.Mode().Perm()&0o077 != 0 {
		file.Close()
		return nil, UnsafePath
	}
	return file, nil
}

func OpenSnapshot(path string) (*os.File, error) {
	if !managedSupported() {
		return nil, Unavailable
	}
	root, err := openSnapshotRoot(filepath.Dir(path), 0)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return openSnapshotFile(root, filepath.Base(path), 0)
}

func transferRecord(ctx context.Context, record *models.Database, state string) (*models.Database, error) {
	if record == nil || record.ID == 0 || record.Kind() != models.EngineRedis || record.ClusterID == "" || record.State != state {
		return nil, IdentityMismatch
	}
	var stored models.Database
	if database.DB.WithContext(ctx).First(&stored, record.ID).Error != nil || stored.Kind() != models.EngineRedis || stored.ClusterID != record.ClusterID || stored.State != state {
		return nil, IdentityMismatch
	}
	return &stored, nil
}

func verifySnapshotIdentity(ctx context.Context, record *models.Database, instance *models.RedisInstance, client *redisclient.Client) error {
	current, err := Instance(ctx, record)
	if err != nil {
		return err
	}
	stored, err := readManifest(current)
	if err != nil {
		return err
	}
	if current.Generation != instance.Generation || current.ProcessID != instance.ProcessID || current.RunID != instance.RunID || current.State != models.DatabaseReady || stored.ProcessID != instance.ProcessID || stored.ProcessStart == "" || processStartIdentity(stored.ProcessID) != stored.ProcessStart {
		return IdentityMismatch
	}
	if err := checkQuarantine(current); err != nil {
		return err
	}
	information, err := client.Info(ctx, "server").Result()
	if err != nil {
		return Unavailable
	}
	fields := informationFields(information)
	if fields["process_id"] != strconv.Itoa(instance.ProcessID) || fields["run_id"] != instance.RunID {
		return IdentityMismatch
	}
	return nil
}

func copySnapshot(ctx context.Context, writer io.Writer, reader io.Reader) (int64, error) {
	buffer := make([]byte, 32*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count, readError := reader.Read(buffer)
		if count > 0 {
			total += int64(count)
			if total > MaximumSnapshotBytes {
				return total, CapacityExceeded
			}
			written, err := writer.Write(buffer[:count])
			if err != nil {
				return total, err
			}
			if written != count {
				return total, io.ErrShortWrite
			}
		}
		if errors.Is(readError, io.EOF) {
			return total, nil
		}
		if readError != nil {
			return total, readError
		}
		if count == 0 {
			return total, io.ErrNoProgress
		}
	}
}

func Snapshot(ctx context.Context, record *models.Database, writer io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, MaximumTransferTimeout)
	defer cancel()
	stored, err := transferRecord(ctx, record, models.DatabaseReady)
	if err != nil || writer == nil {
		return IdentityMismatch
	}
	lock, err := lockOwnedFile(transferLockPath(record.ID), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer lock.Close()
	instance, err := Instance(ctx, stored)
	if err != nil {
		return err
	}
	client, err := privateClient(ctx, stored, instance, systemProfile)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := verifySnapshotIdentity(ctx, stored, instance, client); err != nil {
		return err
	}
	saves, err := beginSnapshot(ctx, client, func(current context.Context) error { return verifySnapshotIdentity(current, stored, instance, client) })
	if err != nil {
		return err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := verifySnapshotIdentity(ctx, stored, instance, client); err != nil {
			return err
		}
		information, err := client.Info(ctx, "persistence").Result()
		if err != nil {
			return Unavailable
		}
		fields := informationFields(information)
		if fields["rdb_bgsave_in_progress"] == "0" {
			completed, err := strconv.ParseUint(fields["rdb_saves"], 10, 64)
			if err != nil || completed != saves+1 || fields["rdb_last_bgsave_status"] != "ok" {
				return Unavailable
			}
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
	userID, _, err := instanceCredentials(record.ID)
	if err != nil {
		return err
	}
	root, err := openSnapshotRoot(dataDirectory(record.ID), userID)
	if err != nil {
		return err
	}
	defer root.Close()
	file, err := openSnapshotFile(root, "dump.rdb", userID)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := verifySnapshotIdentity(ctx, stored, instance, client); err != nil {
		return err
	}
	if _, err := copySnapshot(ctx, writer, file); err != nil {
		return err
	}
	if _, err := transferRecord(ctx, record, models.DatabaseReady); err != nil {
		return err
	}
	return verifySnapshotIdentity(ctx, stored, instance, client)
}
