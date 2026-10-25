package redishost

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func Instance(ctx context.Context, record *models.Database) (*models.RedisInstance, error) {
	if !config.Redis.Enabled || record == nil || record.ID == 0 || record.Kind() != models.EngineRedis || record.ID > maximumDatabaseID {
		return nil, Unavailable
	}
	var instance models.RedisInstance
	if err := database.DB.WithContext(ctx).Where("database_id = ?", record.ID).First(&instance).Error; err != nil {
		return nil, Unavailable
	}
	if instance.Identity == "" || instance.Generation == 0 || record.ClusterID != "" && record.ClusterID != instance.Identity || record.State == models.DatabaseReady && record.ClusterID == "" {
		return nil, IdentityMismatch
	}
	return &instance, nil
}

func prepareInstance(ctx context.Context, record *models.Database) (*models.RedisInstance, error) {
	if record == nil || record.ID == 0 || record.ID > maximumDatabaseID || record.Kind() != models.EngineRedis || record.State == models.DatabaseDeleting {
		return nil, IdentityMismatch
	}
	var instance models.RedisInstance
	err := database.DB.WithContext(ctx).Where("database_id = ?", record.ID).First(&instance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if record.ClusterID != "" || record.State == models.DatabaseReady {
			return nil, IdentityMismatch
		}
		for _, path := range []string{manifestPath(record.ID), configurationDirectory(record.ID), dataDirectory(record.ID), socketDirectory(record.ID)} {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return nil, StorageUnowned
			}
		}
		identity, err := randomSecret()
		if err != nil {
			return nil, err
		}
		instance = models.RedisInstance{DatabaseID: record.ID, Identity: string(identity), Generation: 1, State: models.DatabasePending, MemoryLimit: config.Redis.MemoryLimit, EvictionPolicy: "noeviction", Persistence: "aof-rdb", Fsync: "everysec"}
		clear(identity)
		if err := database.DB.WithContext(ctx).Create(&instance).Error; err != nil {
			return nil, Unavailable
		}
	} else if err != nil {
		return nil, Unavailable
	}
	if record.ClusterID != "" && record.ClusterID != instance.Identity {
		return nil, IdentityMismatch
	}
	if record.ClusterID == "" {
		updated := database.DB.WithContext(ctx).Model(&models.Database{}).Where("id = ? AND engine = ? AND cluster_id = ? AND state IN ?", record.ID, models.EngineRedis, "", []string{models.DatabasePending, models.DatabaseApplying}).Update("cluster_id", instance.Identity)
		if updated.Error != nil || updated.RowsAffected != 1 {
			return nil, IdentityMismatch
		}
		record.ClusterID = instance.Identity
	}
	return &instance, nil
}

func validIdentity(identity string) bool {
	decoded, err := hex.DecodeString(identity)
	return err == nil && len(decoded) == 32
}

func loadManifest(databaseID uint) (*manifest, error) {
	contents, err := readOwnedFile(manifestPath(databaseID))
	if err != nil {
		return nil, err
	}
	var stored manifest
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&stored) != nil || stored.Version != manifestVersion || stored.DatabaseID != databaseID || !validIdentity(stored.Identity) || stored.Generation == 0 || stored.Generation >= math.MaxInt64 {
		return nil, IdentityMismatch
	}
	return &stored, nil
}

func readManifest(instance *models.RedisInstance) (*manifest, error) {
	stored, err := loadManifest(instance.DatabaseID)
	if err != nil {
		return nil, err
	}
	if stored.Identity != instance.Identity || stored.Generation != instance.Generation {
		return nil, IdentityMismatch
	}
	return stored, nil
}

func recoverManifest(ctx context.Context, instance *models.RedisInstance) error {
	stored, err := loadManifest(instance.DatabaseID)
	if errors.Is(err, os.ErrNotExist) && instance.State == models.DatabasePending {
		return nil
	}
	if err != nil {
		return err
	}
	if stored.Identity != instance.Identity || instance.Generation == 0 || instance.Generation >= math.MaxInt64 {
		return IdentityMismatch
	}
	if stored.Generation == instance.Generation {
		return nil
	}
	if stored.Generation+1 != instance.Generation && instance.Generation+1 != stored.Generation {
		return IdentityMismatch
	}
	if stored.ProcessID > 0 && stored.ProcessStart != "" && processStartIdentity(stored.ProcessID) == stored.ProcessStart {
		return PolicyUnconfirmed
	}
	if stored.Generation > instance.Generation {
		updated := database.DB.WithContext(ctx).Model(instance).Where("generation = ? AND identity = ?", instance.Generation, instance.Identity).Updates(map[string]any{"generation": stored.Generation, "state": models.DatabaseApplying, "process_id": 0})
		if updated.Error != nil || updated.RowsAffected != 1 {
			return IdentityMismatch
		}
		instance.Generation = stored.Generation
	}
	instance.ProcessID = 0
	return writeManifest(instance)
}

func writeManifest(instance *models.RedisInstance) error {
	contents, err := json.Marshal(manifest{Version: manifestVersion, DatabaseID: instance.DatabaseID, Identity: instance.Identity, Generation: instance.Generation, ProcessID: instance.ProcessID, ProcessStart: processStartIdentity(instance.ProcessID)})
	if err != nil {
		return IdentityMismatch
	}
	return ownedFile(manifestPath(instance.DatabaseID), contents, 0o600, 0)
}
