package redishost

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"sqldash/database"
	"sqldash/models"
)

type quarantineIntent struct {
	DatabaseID uint
	Identity   string
	Generation uint64
}

func quarantinePath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".blocked")
}

func checkQuarantine(instance *models.RedisInstance) error {
	contents, err := readOwnedFile(quarantinePath(instance.DatabaseID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return PolicyUnconfirmed
	}
	var intent quarantineIntent
	if json.Unmarshal(contents, &intent) != nil || intent.DatabaseID != instance.DatabaseID || !validIdentity(intent.Identity) || intent.Generation == 0 {
		return PolicyUnconfirmed
	}
	if intent.Identity == instance.Identity {
		return PolicyUnconfirmed
	}
	return nil
}

func Quarantine(ctx context.Context, record *models.Database, generation uint64) error {
	if !managedSupported() || record == nil || record.Kind() != models.EngineRedis || record.ID == 0 || !validIdentity(record.ClusterID) || generation == 0 {
		return IdentityMismatch
	}
	stored, err := loadManifest(record.ID)
	if err != nil {
		return PolicyUnconfirmed
	}
	if stored.Identity != record.ClusterID || stored.Generation != generation {
		return nil
	}
	intent, err := json.Marshal(quarantineIntent{DatabaseID: record.ID, Identity: stored.Identity, Generation: generation})
	if err != nil {
		return PolicyUnconfirmed
	}
	if err := ownedFile(quarantinePath(record.ID), intent, 0o600, 0); err != nil {
		return PolicyUnconfirmed
	}
	if stored.ProcessID > 0 && stored.ProcessStart != "" {
		if err := killOwnedProcess(record.ID, stored.ProcessID, stored.ProcessStart); err != nil {
			return PolicyUnconfirmed
		}
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for processStartIdentity(stored.ProcessID) == stored.ProcessStart {
			select {
			case <-ctx.Done():
				return PolicyUnconfirmed
			case <-ticker.C:
			}
		}
	}
	confirmation, cancel := context.WithTimeout(context.WithoutCancel(ctx), 250*time.Millisecond)
	defer cancel()
	_ = database.DB.WithContext(confirmation).Model(&models.RedisInstance{}).Where("database_id = ? AND identity = ? AND generation = ?", record.ID, stored.Identity, generation).Updates(map[string]any{"state": "blocked", "failure": "Redis stopped after access enforcement could not be confirmed", "process_id": 0}).Error
	return nil
}
