package redishost

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"sqldash/models"
)

func beginSnapshot(ctx context.Context, client *redisclient.Client, verify func(context.Context) error) (uint64, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := verify(ctx); err != nil {
			return 0, err
		}
		information, err := client.Info(ctx, "persistence").Result()
		if err != nil {
			return 0, Unavailable
		}
		fields := informationFields(information)
		saves, err := strconv.ParseUint(fields["rdb_saves"], 10, 64)
		if err != nil {
			return 0, Unavailable
		}
		if fields["rdb_bgsave_in_progress"] == "0" && fields["aof_rewrite_in_progress"] == "0" {
			if err := client.BgSave(ctx).Err(); err == nil {
				return saves, nil
			}
			information, err = client.Info(ctx, "persistence").Result()
			if err != nil {
				return 0, Unavailable
			}
			fields = informationFields(information)
			if fields["rdb_bgsave_in_progress"] != "1" && fields["aof_rewrite_in_progress"] != "1" {
				return 0, Unavailable
			}
		} else if fields["rdb_bgsave_in_progress"] != "1" && fields["aof_rewrite_in_progress"] != "1" {
			return 0, Unavailable
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

func verifyInstanceStorage(instance *models.RedisInstance) error {
	userID, _, err := instanceCredentials(instance.DatabaseID)
	if err != nil {
		return err
	}
	for _, entry := range []struct {
		path  string
		owner int
	}{{configurationDirectory(instance.DatabaseID), 0}, {dataDirectory(instance.DatabaseID), userID}} {
		root, err := openSnapshotRoot(entry.path, entry.owner)
		if err != nil {
			return StorageUnowned
		}
		root.Close()
	}
	for _, path := range []string{configurationPath(instance.DatabaseID), aclPath(instance.DatabaseID)} {
		if err := rejectSymlinks(path); err != nil {
			return err
		}
		information, err := os.Lstat(path)
		if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, 0) || information.Mode().Perm() != 0o640 || information.Size() == 0 {
			return StorageUnowned
		}
	}
	path := filepath.Join(dataDirectory(instance.DatabaseID), "dump.rdb")
	minimum := int64(9)
	if instance.Persistence != "rdb" {
		path = filepath.Join(dataDirectory(instance.DatabaseID), "appendonlydir", "appendonly.aof.manifest")
		minimum = 1
	}
	if err := rejectSymlinks(path); err != nil {
		return err
	}
	information, err := os.Lstat(path)
	if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, userID) || information.Size() < minimum {
		return StorageUnowned
	}
	return nil
}
