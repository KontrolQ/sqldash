package redisconnections

import (
	"context"
	"errors"
	"strconv"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redishost"
	"sqldash/services/access"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func Create(ctx context.Context, name string, asked Creation) (*Issued, *fiber.Error) {
	if err := validateCreation(asked); err != nil {
		return nil, failure(err)
	}
	fresh, record, err := authorize(ctx, name, asked.Scope, asked.Subscribe, asked.Publish, false, false)
	if err != nil {
		return nil, failure(err)
	}
	var result *Issued
	err = fenced(fresh, record.ID, func() error {
		fresh, record, err = authorize(fresh, name, asked.Scope, asked.Subscribe, asked.Publish, false, false)
		if err != nil {
			return err
		}
		instance, err := redishost.Instance(fresh, record)
		if err != nil {
			return Unavailable
		}
		var count int64
		if err := database.DB.WithContext(fresh).Model(&models.RedisConnection{}).Where("database_id = ? AND revoked_at IS NULL", record.ID).Count(&count).Error; err != nil {
			return Unavailable
		}
		if count >= maximumCredentials {
			return Conflict
		}
		suffix, err := randomValue(12)
		if err != nil {
			return err
		}
		connection := &models.RedisConnection{DatabaseID: record.ID, InstanceIdentity: instance.Identity, AccountID: access.AccountID(fresh), Username: "sqd_" + strconv.FormatUint(uint64(record.ID), 10) + "_" + suffix, Label: asked.Label, Scope: asked.Scope, Subscribe: asked.Subscribe, Publish: asked.Publish, ExpiresAt: asked.ExpiresAt, Generation: 1}
		if err := stagePassword(fresh, record, connection, instance); err != nil {
			return err
		}
		if err := reconcileCredential(fresh, record, connection, instance); err != nil {
			return err
		}
		result, err = deliver(fresh, record, connection, instance)
		return err
	})
	return result, failure(err)
}

func Rotate(ctx context.Context, name string, identifier uint) (*Issued, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, models.ScopeReadOnly, false, false, false, false)
	if err != nil {
		return nil, failure(err)
	}
	var result *Issued
	err = fenced(fresh, record.ID, func() error {
		connection, err := loadConnection(fresh, record.ID, identifier)
		if err != nil {
			return err
		}
		fresh, record, err = authorize(fresh, name, connection.Scope, connection.Subscribe, connection.Publish, connection.PrimaryConnection, false)
		if err != nil {
			return err
		}
		if connection.ExpiresAt != nil && !connection.ExpiresAt.After(time.Now()) {
			return Invalid
		}
		instance, err := redishost.Instance(fresh, record)
		if err != nil || instance.Identity != connection.InstanceIdentity {
			return Conflict
		}
		if err := clearPending(fresh, connection, connection.State); err != nil {
			return err
		}
		connection.AccountID = access.AccountID(fresh)
		if err := stagePassword(fresh, record, connection, instance); err != nil {
			return err
		}
		if err := reconcileCredential(fresh, record, connection, instance); err != nil {
			return err
		}
		result, err = deliver(fresh, record, connection, instance)
		return err
	})
	return result, failure(err)
}

func Revoke(ctx context.Context, name string, identifier uint) *fiber.Error {
	fresh, record, err := authorize(ctx, name, models.ScopeReadOnly, false, false, false, false)
	if err != nil {
		return failure(err)
	}
	err = fenced(fresh, record.ID, func() error {
		connection, err := loadConnection(fresh, record.ID, identifier)
		if err != nil {
			return err
		}
		fresh, record, err = authorize(fresh, name, connection.Scope, connection.Subscribe, connection.Publish, connection.PrimaryConnection, false)
		if err != nil {
			return err
		}
		instance, err := redishost.Instance(fresh, record)
		if err != nil || instance.Identity != connection.InstanceIdentity {
			return Conflict
		}
		if connection.RevokedAt == nil {
			if connection.Generation == 0 || connection.Generation >= 1<<63-1 {
				return Conflict
			}
			now := time.Now().UTC()
			updated := database.DB.WithContext(fresh).Model(&models.RedisConnection{}).Where("id = ? AND generation = ? AND state = ?", connection.ID, connection.Generation, connection.State).Updates(map[string]any{"generation": connection.Generation + 1, "revoked_at": now, "state": stateRevoked})
			if updated.Error != nil || updated.RowsAffected != 1 {
				return Conflict
			}
			connection.Generation++
			connection.RevokedAt = &now
			connection.State = stateRevoked
		}
		return suspend(fresh, record, connection, instance, stateRevoked)
	})
	return failure(err)
}

func Reveal(ctx context.Context, name string, identifier uint) (*Issued, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, models.ScopeReadOnly, false, false, false, false)
	if err != nil {
		return nil, failure(err)
	}
	var result *Issued
	err = fenced(fresh, record.ID, func() error {
		connection, err := loadConnection(fresh, record.ID, identifier)
		if err != nil {
			return err
		}
		fresh, record, err = authorize(fresh, name, connection.Scope, connection.Subscribe, connection.Publish, connection.PrimaryConnection, false)
		if err != nil {
			return err
		}
		if connection.AccountID != access.AccountID(fresh) {
			return Refused
		}
		instance, err := redishost.Instance(fresh, record)
		if err != nil || instance.Identity != connection.InstanceIdentity {
			return Conflict
		}
		result, err = deliver(fresh, record, connection, instance)
		return err
	})
	return result, failure(err)
}

func EnsurePrimary(ctx context.Context, record *models.Database, accountID uint) error {
	if record == nil || record.ID == 0 || record.Kind() != models.EngineRedis || accountID == 0 {
		return Refused
	}
	actor, err := access.LoadWith(database.DB.WithContext(ctx), accountID)
	if err != nil {
		return Refused
	}
	fresh, current, err := authorize(access.WithContext(ctx, actor), record.Name, models.ScopeReadWrite, false, false, true, true)
	if err != nil || current.ID != record.ID || current.ClusterID != record.ClusterID {
		return Refused
	}
	return fenced(fresh, current.ID, func() error {
		fresh, current, err = authorize(fresh, record.Name, models.ScopeReadWrite, false, false, true, true)
		if err != nil {
			return err
		}
		instance, err := redishost.Instance(fresh, current)
		if err != nil {
			return Unavailable
		}
		var connection models.RedisConnection
		err = database.DB.WithContext(fresh).Where("database_id = ? AND primary_connection = ?", current.ID, true).First(&connection).Error
		if err == nil {
			if connection.InstanceIdentity == instance.Identity {
				return reconcileCredential(fresh, current, &connection, instance)
			}
			if err := reconcileCredential(fresh, current, &connection, instance); err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return Unavailable
		}
		if !models.PublicRedisUsername(current.Name) {
			return Conflict
		}
		connection = models.RedisConnection{DatabaseID: current.ID, InstanceIdentity: instance.Identity, AccountID: accountID, Username: current.Name, Label: "Primary", Scope: models.ScopeReadWrite, PrimaryConnection: true, Subscribe: access.FromContext(fresh).Can(permissions.RedisSubscribe), Publish: access.FromContext(fresh).Can(permissions.RedisPublish), Generation: 1}
		if err := stagePassword(fresh, current, &connection, instance); err != nil {
			return err
		}
		return reconcileCredential(fresh, current, &connection, instance)
	})
}

func RotatePrimary(ctx context.Context, name string) (*Issued, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, models.ScopeReadWrite, false, false, true, false)
	if err != nil {
		return nil, failure(err)
	}
	var primary models.RedisConnection
	if err := database.DB.WithContext(fresh).Where("database_id = ? AND primary_connection = ?", record.ID, true).First(&primary).Error; err != nil {
		return nil, failure(Missing)
	}
	return Rotate(fresh, name, primary.ID)
}
