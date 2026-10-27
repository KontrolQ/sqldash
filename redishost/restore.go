package redishost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/rediswire"
)

type restoreIntent struct {
	DatabaseID   uint
	Identity     string
	Generation   uint64
	Digest       string
	Token        string
	Phase        string
	ProcessID    int
	ProcessStart string
}

func restoreIntentPath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".restore.json")
}
func restoreWorkPath(intent *restoreIntent) string {
	return filepath.Join(configurationDirectory(intent.DatabaseID), "restore-"+intent.Token)
}
func restoreBackupPath(intent *restoreIntent) string {
	return dataDirectory(intent.DatabaseID) + ".restore-" + intent.Token
}

func readRestoreIntent(instance *models.RedisInstance) (*restoreIntent, error) {
	contents, err := readOwnedFile(restoreIntentPath(instance.DatabaseID))
	if err != nil {
		return nil, err
	}
	var intent restoreIntent
	if json.Unmarshal(contents, &intent) != nil || intent.DatabaseID != instance.DatabaseID || intent.Identity != instance.Identity || intent.Generation == 0 || !validIdentity(intent.Digest) || len(intent.Token) != 16 {
		return nil, IdentityMismatch
	}
	if _, err := hex.DecodeString(intent.Token); err != nil {
		return nil, IdentityMismatch
	}
	switch intent.Phase {
	case "staging", "installing", "installed", "completed", "capacity":
	default:
		return nil, IdentityMismatch
	}
	return &intent, nil
}
func writeRestoreIntent(intent *restoreIntent) error {
	contents, err := json.Marshal(intent)
	if err != nil {
		return Unavailable
	}
	return ownedFile(restoreIntentPath(intent.DatabaseID), contents, 0o600, 0)
}
func restorePending(ctx context.Context, record *models.Database) bool {
	instance, err := Instance(ctx, record)
	if err != nil {
		return false
	}
	intent, err := readRestoreIntent(instance)
	return !errors.Is(err, os.ErrNotExist) && (err != nil || intent.Phase != "completed")
}
func restoreTarget(ctx context.Context, record *models.Database) (*models.Database, *models.RedisInstance, error) {
	stored, err := transferRecord(ctx, record, models.DatabaseApplying)
	if err != nil {
		return nil, nil, err
	}
	instance, err := Instance(ctx, stored)
	if err != nil {
		return nil, nil, err
	}
	if stored.Protected || instance.Published {
		return nil, nil, PolicyUnconfirmed
	}
	if err := checkQuarantine(instance); err != nil {
		return nil, nil, err
	}
	if _, err := readManifest(instance); err != nil {
		return nil, nil, err
	}
	return stored, instance, nil
}
func validRestoreToken(token string) bool {
	if len(token) < 9 || len(token) > 96 || !strings.HasPrefix(token, "restore-") {
		return false
	}
	for _, character := range token {
		if character != '-' && (character < '0' || character > '9') && (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') {
			return false
		}
	}
	return true
}
func snapshotDigest(ctx context.Context, file *os.File) (string, error) {
	var header [9]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return "", InvalidSettings
	}
	if string(header[:5]) != "REDIS" {
		return "", InvalidSettings
	}
	if _, err := strconv.ParseUint(string(header[5:]), 10, 16); err != nil {
		return "", InvalidSettings
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", Unavailable
	}
	digest := sha256.New()
	if _, err := copySnapshot(ctx, digest, file); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", Unavailable
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func Clone(ctx context.Context, source, target *models.Database) error {
	ctx, cancel := context.WithTimeout(ctx, MaximumTransferTimeout)
	defer cancel()
	if source == nil || target == nil || source.ID == target.ID {
		return IdentityMismatch
	}
	if _, _, err := restoreTarget(ctx, target); err != nil {
		return err
	}
	lock, err := lockOwnedFile(cloneLockPath(target.ID), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer lock.Close()
	file, err := cachedCloneSnapshot(ctx, source, target)
	if err != nil {
		return err
	}
	defer file.Close()
	return Restore(ctx, target, file)
}

func Restore(ctx context.Context, record *models.Database, reader io.Reader) error {
	ctx, cancel := context.WithTimeout(ctx, MaximumTransferTimeout)
	defer cancel()
	_, instance, err := restoreTarget(ctx, record)
	if err != nil {
		return err
	}
	if reader == nil {
		return InvalidSettings
	}
	root, err := openSnapshotRoot(config.RedisStagingPath(), 0)
	if err != nil {
		return err
	}
	defer root.Close()
	token, err := randomSecret()
	if err != nil {
		return err
	}
	defer clear(token)
	name := "restore-" + string(token)
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return Unavailable
	}
	defer file.Close()
	defer root.Remove(name)
	if _, err := copySnapshot(ctx, file, reader); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return Unavailable
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Unavailable
	}
	if _, err := snapshotDigest(ctx, file); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return Unavailable
	}
	_, err = requestControl(ctx, controlRequest{Action: "restore", DatabaseID: record.ID, Identity: record.ClusterID, Generation: instance.Generation, RestoreToken: name})
	return err
}

func (self *manager) restore(ctx context.Context, record *models.Database, generation uint64, token string) (result error) {
	ctx, cancel := context.WithTimeout(ctx, MaximumTransferTimeout)
	defer cancel()
	stored, instance, err := restoreTarget(ctx, record)
	if err != nil {
		return err
	}
	if generation != instance.Generation || !validRestoreToken(token) {
		return IdentityMismatch
	}
	policy, err := lockOwnedFile(stateLockPath(record.ID), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer policy.Close()
	upload, err := OpenSnapshot(filepath.Join(config.RedisStagingPath(), token))
	if err != nil {
		return err
	}
	defer upload.Close()
	digest, err := snapshotDigest(ctx, upload)
	if err != nil {
		return err
	}
	intent, err := readRestoreIntent(instance)
	if errors.Is(err, os.ErrNotExist) {
		control, err := privateClient(ctx, stored, instance, systemProfile)
		if err != nil {
			return err
		}
		err = verifySnapshotIdentity(ctx, stored, instance, control)
		keys, keyError := control.DBSize(ctx).Result()
		functions, functionError := control.Do(ctx, "FUNCTION", "LIST").Slice()
		indexes, indexError := control.Do(ctx, "FT._LIST").StringSlice()
		control.Close()
		if err != nil {
			return err
		}
		if keyError != nil || functionError != nil || indexError != nil || keys != 0 || len(functions) != 0 || len(indexes) != 0 {
			return StorageUnowned
		}
		secret, err := randomSecret()
		if err != nil {
			return err
		}
		intent = &restoreIntent{DatabaseID: record.ID, Identity: instance.Identity, Generation: instance.Generation, Digest: digest, Token: string(secret[:16]), Phase: "staging"}
		clear(secret)
		if err := writeRestoreIntent(intent); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if intent.Digest != digest {
		return IdentityMismatch
	}
	if intent.Phase == "capacity" {
		return CapacityExceeded
	}
	if intent.Phase == "completed" {
		control, err := privateClient(ctx, stored, instance, systemProfile)
		if err != nil {
			return err
		}
		defer control.Close()
		return verifySnapshotIdentity(ctx, stored, instance, control)
	}
	defer func() {
		if result != nil {
			cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), config.Redis.ShutdownTimeout+time.Second)
			defer stop()
			_ = self.stop(stored, false)
			_ = database.DB.WithContext(cleanup).Model(&models.RedisInstance{}).Where("database_id = ? AND identity = ?", record.ID, instance.Identity).Updates(map[string]any{"state": "blocked", "failure": "Redis restore is incomplete; retry the same snapshot", "process_id": 0}).Error
		}
	}()
	if err := database.DB.WithContext(ctx).Model(instance).Update("state", "restoring").Error; err != nil {
		return Unavailable
	}
	if err := self.stop(stored, false); err != nil {
		return err
	}
	if intent.ProcessID > 0 && intent.ProcessStart != "" {
		if err := killOwnedProcess(record.ID, intent.ProcessID, intent.ProcessStart); err != nil {
			return err
		}
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for processStartIdentity(intent.ProcessID) == intent.ProcessStart {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	}
	ownership, err := lockOwnedFile(lockPath(record.ID), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer ownership.Close()
	instance, err = Instance(ctx, stored)
	if err != nil {
		return err
	}
	if err := recoverManifest(ctx, instance); err != nil {
		return err
	}
	if err := database.DB.WithContext(ctx).Model(instance).Updates(map[string]any{"state": "restoring", "process_id": 0}).Error; err != nil {
		return Unavailable
	}
	if intent.Phase == "staging" {
		if err := self.stageRestore(ctx, stored, instance, intent, upload, ownership); err != nil {
			return err
		}
		intent.Phase = "installing"
		intent.ProcessID = 0
		intent.ProcessStart = ""
		if err := writeRestoreIntent(intent); err != nil {
			return err
		}
	}
	if _, _, err := restoreTarget(ctx, stored); err != nil {
		return err
	}
	if intent.Phase == "installing" {
		if err := installRestore(ctx, instance, intent); err != nil {
			return err
		}
		intent.Phase = "installed"
		if err := writeRestoreIntent(intent); err != nil {
			return err
		}
	}
	if err := ownership.Close(); err != nil {
		return Unavailable
	}
	if err := self.start(ctx, stored); err != nil {
		return err
	}
	_, started, err := restoreTarget(ctx, stored)
	if err != nil {
		return err
	}
	control, err := privateClient(ctx, stored, started, systemProfile)
	if err != nil {
		return err
	}
	defer control.Close()
	if err := verifySnapshotIdentity(ctx, stored, started, control); err != nil {
		return err
	}
	capacityError := restoreCapacity(ctx, control, started)
	statistics, err := control.Info(ctx, "stats").Result()
	if err != nil {
		return Unavailable
	}
	evicted, err := strconv.ParseUint(informationFields(statistics)["evicted_keys"], 10, 64)
	if err != nil {
		return Unavailable
	}
	if evicted != 0 {
		intent.Phase = "capacity"
		if err := writeRestoreIntent(intent); err != nil {
			return err
		}
		return CapacityExceeded
	}
	if capacityError != nil {
		return capacityError
	}
	intent.Phase = "completed"
	if err := writeRestoreIntent(intent); err != nil {
		return err
	}
	_ = removeRestoreTree(restoreBackupPath(intent), instance.DatabaseID)
	_ = removeRestoreWork(intent)
	return nil
}

func removeRestoreState(record *models.Database) error {
	intent, err := readRestoreIntent(&models.RedisInstance{DatabaseID: record.ID, Identity: record.ClusterID})
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := removeRestoreTree(restoreBackupPath(intent), record.ID); err != nil {
		return err
	}
	return os.Remove(restoreIntentPath(record.ID))
}

func syncRestoreDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return Unavailable
	}
	defer directory.Close()
	return directory.Sync()
}
func inspectRestoreTree(path string, databaseID uint) error {
	userID, _, err := instanceCredentials(databaseID)
	if err != nil {
		return err
	}
	root, err := openSnapshotRoot(path, userID)
	if err != nil {
		return err
	}
	defer root.Close()
	var files int
	var size int64
	var inspect func(string, int) error
	inspect = func(name string, depth int) error {
		if depth > 64 {
			return CapacityExceeded
		}
		information, err := root.Lstat(name)
		if err != nil {
			return UnsafePath
		}
		files++
		size += information.Size()
		if files > 4096 || size > MaximumSnapshotBytes*2 {
			return CapacityExceeded
		}
		socket := filepath.Base(name) == "restore.sock" && information.Mode()&os.ModeSocket != 0
		if !verifyOwner(information, userID) || !information.IsDir() && !information.Mode().IsRegular() && !socket {
			return UnsafePath
		}
		if !information.IsDir() {
			return nil
		}
		directory, err := root.Open(name)
		if err != nil {
			return UnsafePath
		}
		defer directory.Close()
		opened, err := directory.Stat()
		if err != nil || !os.SameFile(information, opened) {
			return UnsafePath
		}
		for {
			entries, readError := directory.ReadDir(64)
			for _, entry := range entries {
				if err := inspect(filepath.Join(name, entry.Name()), depth+1); err != nil {
					return err
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
	return inspect(".", 0)
}
func removeRestoreTree(path string, databaseID uint) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := inspectRestoreTree(path, databaseID); err != nil {
		return err
	}
	return os.RemoveAll(path)
}
func removeRestoreWork(intent *restoreIntent) error {
	path := restoreWorkPath(intent)
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := rejectSymlinks(path); err != nil {
		return err
	}
	information, err := os.Lstat(path)
	if err != nil || !information.IsDir() || !verifyOwner(information, 0) {
		return UnsafePath
	}
	if err := removeRestoreTree(filepath.Join(path, "data"), intent.DatabaseID); err != nil {
		return err
	}
	return os.Remove(path)
}
func (self *manager) stageRestore(ctx context.Context, record *models.Database, instance *models.RedisInstance, intent *restoreIntent, upload *os.File, ownership *os.File) error {
	if err := removeRestoreWork(intent); err != nil {
		return err
	}
	userID, groupID, err := instanceCredentials(record.ID)
	if err != nil {
		return err
	}
	work := restoreWorkPath(intent)
	data := filepath.Join(work, "data")
	if err := ownedDirectory(work, 0o710, 0, groupID); err != nil {
		return err
	}
	if err := ownedDirectory(data, 0o700, userID, groupID); err != nil {
		return err
	}
	root, err := openSnapshotRoot(data, userID)
	if err != nil {
		return err
	}
	defer root.Close()
	file, err := root.OpenFile("dump.rdb", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return Unavailable
	}
	if _, err := upload.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return Unavailable
	}
	_, copyError := copySnapshot(ctx, file, upload)
	ownershipError := file.Chown(userID, groupID)
	syncError := file.Sync()
	closeError := file.Close()
	if err := errors.Join(copyError, ownershipError, syncError, closeError); err != nil {
		return err
	}
	if err := syncRestoreDirectory(data); err != nil {
		return err
	}
	if err := bootstrapRestore(ctx, record, instance, intent, ownership); err != nil {
		return err
	}
	if err := syncRestoreDirectory(work); err != nil {
		return err
	}
	return syncRestoreDirectory(configurationDirectory(record.ID))
}

func restoreCapacity(ctx context.Context, control *redisclient.Client, instance *models.RedisInstance) error {
	memory, err := control.Info(ctx, "memory").Result()
	if err != nil {
		return Unavailable
	}
	fields := informationFields(memory)
	used, usedError := strconv.ParseInt(fields["used_memory"], 10, 64)
	excluded, excludedError := strconv.ParseInt(fields["mem_not_counted_for_evict"], 10, 64)
	if usedError != nil || excludedError != nil || used < 0 || excluded < 0 || excluded > used {
		return Unavailable
	}
	if !restoreMemoryAllowed(used, excluded, instance.MemoryLimit) {
		return CapacityExceeded
	}
	return nil
}

func bootstrapRestore(ctx context.Context, record *models.Database, instance *models.RedisInstance, intent *restoreIntent, ownership *os.File) (result error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	data := filepath.Join(restoreWorkPath(intent), "data")
	socket := filepath.Join(data, "restore.sock")
	if len(socket) > 100 {
		return UnsafePath
	}
	log, err := processLog(record.ID)
	if err != nil {
		return err
	}
	command := exec.Command(config.Redis.BinaryPath, configurationPath(record.ID), "--dir", data, "--unixsocket", socket, "--appendonly", "no", "--save", "", "--appendfsync", "always", "--maxmemory-policy", "noeviction")
	command.Dir = data
	command.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8", "HOME=/nonexistent"}
	command.Stdout = log
	command.Stderr = log
	if err := configureProcess(command, record.ID, ownership); err != nil {
		log.file.Close()
		return err
	}
	if err := command.Start(); err != nil {
		log.file.Close()
		return Unavailable
	}
	finished := make(chan struct{})
	var processError error
	go func() { processError = command.Wait(); log.file.Close(); close(finished) }()
	stopped := false
	stop := func() error {
		if stopped {
			return nil
		}
		stopped = true
		select {
		case <-finished:
			return processError
		default:
		}
		_ = command.Process.Signal(syscall.SIGTERM)
		select {
		case <-finished:
			return processError
		case <-time.After(config.Redis.ShutdownTimeout):
		}
		_ = command.Process.Kill()
		select {
		case <-finished:
			return PolicyUnconfirmed
		case <-time.After(5 * time.Second):
			return PolicyUnconfirmed
		}
	}
	defer func() { result = errors.Join(result, stop()) }()
	intent.ProcessID = command.Process.Pid
	intent.ProcessStart = processStartIdentity(command.Process.Pid)
	if intent.ProcessStart == "" {
		return IdentityMismatch
	}
	if err := writeRestoreIntent(intent); err != nil {
		return err
	}
	password, err := systemPassword(instance, systemProfile, false)
	if err != nil {
		return err
	}
	control := redisclient.NewClient(&redisclient.Options{
		Addr: socket, Network: "unix", Protocol: 2, Username: systemUsername(record.ID, systemProfile), Password: string(password), DisableIdentity: true, MaxRetries: -1, PoolSize: 1,
		DialTimeout: controlTimeout, ReadTimeout: config.Redis.CommandTimeout, WriteTimeout: controlTimeout, ContextTimeoutEnabled: true,
		Dialer: func(parent context.Context, network, address string) (net.Conn, error) {
			if err := rejectSymlinks(socket); err != nil {
				return nil, err
			}
			information, err := os.Lstat(socket)
			userID, _, credentialError := instanceCredentials(record.ID)
			if err != nil || credentialError != nil || information.Mode()&os.ModeSocket == 0 || !verifyOwner(information, userID) || information.Mode().Perm() != 0o600 {
				return nil, Unavailable
			}
			connection, err := (&net.Dialer{Timeout: controlTimeout}).DialContext(parent, "unix", socket)
			if err != nil {
				return nil, Unavailable
			}
			if !socketPeerMatches(connection, record.ID, command.Process.Pid) {
				connection.Close()
				return nil, IdentityMismatch
			}
			return rediswire.Guard(connection, rediswire.Limits{MaximumBytes: 1024 * 1024, MaximumBulkBytes: 1024 * 1024, MaximumElements: 1024, MaximumDepth: 8}), nil
		},
	})
	clear(password)
	defer control.Close()
	startup, cancel := context.WithTimeout(ctx, config.Redis.StartupTimeout)
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for control.Ping(startup).Err() != nil {
		select {
		case <-startup.Done():
			return Unavailable
		case <-finished:
			return InvalidSettings
		case <-ticker.C:
		}
	}
	information, err := control.Info(ctx, "server").Result()
	if err != nil {
		return Unavailable
	}
	server := informationFields(information)
	runID := server["run_id"]
	if server["process_id"] != strconv.Itoa(command.Process.Pid) || runID == "" {
		return IdentityMismatch
	}
	if err := restoreCapacity(ctx, control, instance); err != nil {
		return err
	}
	if instance.Persistence != "rdb" {
		if err := control.ConfigSet(ctx, "appendonly", "yes").Err(); err != nil {
			return Unavailable
		}
		for {
			if _, _, err := restoreTarget(ctx, record); err != nil {
				return err
			}
			if err := inspectRestoreTree(data, record.ID); err != nil {
				return err
			}
			persistence, err := control.Info(ctx, "persistence").Result()
			if err != nil {
				return Unavailable
			}
			fields := informationFields(persistence)
			if fields["aof_enabled"] == "1" && fields["aof_rewrite_in_progress"] == "0" && fields["aof_rewrite_scheduled"] == "0" {
				size, err := strconv.ParseInt(fields["aof_base_size"], 10, 64)
				if err != nil || size == 0 || fields["aof_last_bgrewrite_status"] != "ok" {
					return InvalidSettings
				}
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-finished:
				return InvalidSettings
			case <-ticker.C:
			}
		}
	}
	if err := restoreCapacity(ctx, control, instance); err != nil {
		return err
	}
	information, err = control.Info(ctx, "server").Result()
	if err != nil {
		return Unavailable
	}
	if fields := informationFields(information); fields["run_id"] != runID || fields["process_id"] != strconv.Itoa(command.Process.Pid) || processStartIdentity(command.Process.Pid) != intent.ProcessStart {
		return IdentityMismatch
	}
	if err := stop(); err != nil {
		return err
	}
	if information, err := os.Lstat(socket); err == nil {
		if information.Mode()&os.ModeSocket == 0 {
			return UnsafePath
		}
		if err := os.Remove(socket); err != nil {
			return UnsafePath
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return UnsafePath
	}
	if err := inspectRestoreTree(data, record.ID); err != nil {
		return err
	}
	return syncRestoreDirectory(data)
}

func installRestore(ctx context.Context, instance *models.RedisInstance, intent *restoreIntent) error {
	staged := filepath.Join(restoreWorkPath(intent), "data")
	target := dataDirectory(instance.DatabaseID)
	backup := restoreBackupPath(intent)
	targetInfo, targetError := os.Lstat(target)
	_, backupError := os.Lstat(backup)
	_, stagedError := os.Lstat(staged)
	targetExists := targetError == nil
	backupExists := backupError == nil
	stagedExists := stagedError == nil
	for _, err := range []error{targetError, backupError, stagedError} {
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return UnsafePath
		}
	}
	if !backupExists {
		if !targetExists || !stagedExists {
			return StorageUnowned
		}
		if err := inspectRestoreTree(target, instance.DatabaseID); err != nil {
			return err
		}
		if !targetInfo.IsDir() {
			return UnsafePath
		}
		if err := os.Rename(target, backup); err != nil {
			return Unavailable
		}
		if err := syncRestoreDirectory(config.RedisInstancesPath()); err != nil {
			return err
		}
		targetExists = false
		backupExists = true
	}
	if !targetExists {
		if !stagedExists {
			return StorageUnowned
		}
		if err := inspectRestoreTree(staged, instance.DatabaseID); err != nil {
			return err
		}
		if err := os.Rename(staged, target); err != nil {
			return Unavailable
		}
		if err := syncRestoreDirectory(config.RedisInstancesPath()); err != nil {
			return err
		}
		if err := syncRestoreDirectory(restoreWorkPath(intent)); err != nil {
			return err
		}
	} else if stagedExists {
		return StorageUnowned
	}
	if !backupExists {
		return StorageUnowned
	}
	userID, _, err := instanceCredentials(instance.DatabaseID)
	if err != nil {
		return err
	}
	root, err := openSnapshotRoot(target, userID)
	if err != nil {
		return err
	}
	defer root.Close()
	file, err := openSnapshotFile(root, "dump.rdb", userID)
	if err != nil {
		return err
	}
	defer file.Close()
	digest, err := snapshotDigest(ctx, file)
	if err != nil {
		return err
	}
	if digest != intent.Digest {
		return IdentityMismatch
	}
	return inspectRestoreTree(target, instance.DatabaseID)
}
