package redisconnections

import (
	"context"
	"errors"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/redishost"

	"gorm.io/gorm"
)

func quarantine(ctx context.Context, record *models.Database, instance *models.RedisInstance) error {
	confirmation, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	_ = redishost.Quarantine(confirmation, record, instance.Generation)
	return Unavailable
}

func reconcileCredential(ctx context.Context, record *models.Database, connection *models.RedisConnection, instance *models.RedisInstance) error {
	if connection.InstanceIdentity != instance.Identity {
		retiredState := stateStale
		if connection.RevokedAt != nil {
			retiredState = stateRevoked
		}
		if err := suspend(ctx, record, connection, instance, retiredState); err != nil {
			return err
		}
		if connection.PrimaryConnection {
			return retirePrimary(ctx, record, connection)
		}
		return nil
	}
	if err := renamePrimary(ctx, record, connection, instance); err != nil {
		return err
	}
	if connection.DeletedAt.Valid || connection.RevokedAt != nil || connection.ExpiresAt != nil && !time.Now().Before(*connection.ExpiresAt) {
		return suspend(ctx, record, connection, instance, stateRevoked)
	}
	if connection.State == stateStale {
		return suspend(ctx, record, connection, instance, stateStale)
	}
	valid, err := issuerAuthorized(ctx, record, connection)
	if err != nil {
		return err
	}
	if !valid {
		return suspend(ctx, record, connection, instance, stateSuspended)
	}
	if err := upgradePrimaryPubSub(ctx, record, connection, instance); err != nil {
		return err
	}
	if connection.State == stateDelivering {
		if err := clearPending(ctx, connection, models.DatabaseReady); err != nil {
			return err
		}
	} else if connection.PendingSecret != "" {
		if _, err := loadPending(connection, instance); err != nil {
			if errors.Is(err, Consumed) {
				if _, retainedError := recoverPrimary(connection, instance); retainedError != nil {
					return suspend(ctx, record, connection, instance, stateStale)
				}
				if err := clearPending(ctx, connection, connection.State); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	} else if connection.State == models.DatabasePending {
		return suspend(ctx, record, connection, instance, stateStale)
	}
	profile := redishost.ReadOnly
	if connection.Scope == models.ScopeReadWrite {
		profile = redishost.ReadWrite
	}
	if err := redishost.SetVerifier(ctx, record, connection.Username, connection.PasswordDigest, profile, connection.Subscribe, connection.Publish); err != nil {
		return quarantine(ctx, record, instance)
	}
	currentInstance, err := redishost.Instance(ctx, record)
	if err != nil || currentInstance.Identity != instance.Identity || currentInstance.Generation != instance.Generation {
		return Conflict
	}
	valid, err = issuerAuthorized(ctx, record, connection)
	if err != nil {
		return err
	}
	if !valid {
		return suspend(ctx, record, connection, instance, stateSuspended)
	}
	if connection.State != models.DatabaseReady {
		updated := database.DB.WithContext(ctx).Model(&models.RedisConnection{}).Where("id = ? AND generation = ? AND state = ? AND instance_identity = ? AND revoked_at IS NULL", connection.ID, connection.Generation, connection.State, instance.Identity).Update("state", models.DatabaseReady)
		if updated.Error != nil || updated.RowsAffected != 1 {
			return Conflict
		}
		connection.State = models.DatabaseReady
	}
	return nil
}

func suspend(ctx context.Context, record *models.Database, connection *models.RedisConnection, instance *models.RedisInstance, state string) error {
	if err := redishost.RemoveUser(ctx, record, connection.Username); err != nil {
		return quarantine(ctx, record, instance)
	}
	if err := clearPending(ctx, connection, state); err != nil {
		return err
	}
	if state == stateRevoked || state == stateStale {
		if err := clearPrimary(connection); err != nil {
			return err
		}
	}
	if connection.State != state {
		updated := database.DB.WithContext(ctx).Unscoped().Model(&models.RedisConnection{}).Where("id = ? AND generation = ? AND state = ?", connection.ID, connection.Generation, connection.State).Update("state", state)
		if updated.Error != nil || updated.RowsAffected != 1 {
			return Conflict
		}
		connection.State = state
	}
	return nil
}

func Reconcile(ctx context.Context, record *models.Database) error {
	if record == nil || record.ID == 0 || record.Kind() != models.EngineRedis || record.ClusterID == "" {
		return Refused
	}
	return fenced(ctx, record.ID, func() error {
		var current models.Database
		if err := database.DB.WithContext(ctx).First(&current, record.ID).Error; err != nil || current.Kind() != models.EngineRedis || current.ClusterID != record.ClusterID || current.State != models.DatabaseReady && current.State != models.DatabaseApplying {
			return Refused
		}
		instance, err := redishost.Instance(ctx, &current)
		if err != nil {
			return Unavailable
		}
		var connections []models.RedisConnection
		return database.DB.WithContext(ctx).Unscoped().Where("database_id = ?", current.ID).Order("id asc").FindInBatches(&connections, 100, func(_ *gorm.DB, _ int) error {
			for index := range connections {
				if err := reconcileCredential(ctx, &current, &connections[index], instance); err != nil {
					return err
				}
			}
			return nil
		}).Error
	})
}
