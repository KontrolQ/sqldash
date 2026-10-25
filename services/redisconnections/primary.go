package redisconnections

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/redishost"
	"sqldash/secretstore"

	"github.com/gofiber/fiber/v3"
)

const primaryNamespace = "redis-primary"

type primaryPassword struct {
	DatabaseID       uint
	CredentialID     uint
	InstanceIdentity string
	Generation       uint64
	Password         string
}

func primarySecretName(identifier uint) string {
	return "c" + strconv.FormatUint(uint64(identifier), 10)
}

func retainPrimary(connection *models.RedisConnection, password string) error {
	if !connection.PrimaryConnection {
		return nil
	}
	if len(password) != 64 || digestPassword(password) != connection.PasswordDigest {
		return Conflict
	}
	store, err := redishost.SecretStore()
	if err != nil {
		return Unavailable
	}
	encoded, err := json.Marshal(primaryPassword{DatabaseID: connection.DatabaseID, CredentialID: connection.ID, InstanceIdentity: connection.InstanceIdentity, Generation: connection.Generation, Password: password})
	if err != nil {
		return Unavailable
	}
	defer clear(encoded)
	if err := store.Set(primaryNamespace, primarySecretName(connection.ID), encoded); err != nil {
		return Unavailable
	}
	return nil
}

func retainedPrimary(connection *models.RedisConnection, instance *models.RedisInstance) (string, error) {
	if !connection.PrimaryConnection || connection.InstanceIdentity != instance.Identity {
		return "", Conflict
	}
	store, err := redishost.SecretStore()
	if err != nil {
		return "", Unavailable
	}
	encoded, err := store.Get(primaryNamespace, primarySecretName(connection.ID))
	if errors.Is(err, secretstore.ErrNotFound) {
		return "", Consumed
	}
	if err != nil {
		return "", Unavailable
	}
	defer clear(encoded)
	var held primaryPassword
	if json.Unmarshal(encoded, &held) != nil || held.DatabaseID != connection.DatabaseID || held.CredentialID != connection.ID || held.InstanceIdentity != connection.InstanceIdentity || held.Generation != connection.Generation || len(held.Password) != 64 || digestPassword(held.Password) != connection.PasswordDigest {
		return "", Conflict
	}
	return held.Password, nil
}

func recoverPrimary(connection *models.RedisConnection, instance *models.RedisInstance) (string, error) {
	password, err := retainedPrimary(connection, instance)
	if !errors.Is(err, Consumed) {
		return password, err
	}
	if !connection.PrimaryConnection || connection.PendingSecret == "" || connection.InstanceIdentity != instance.Identity {
		return "", Consumed
	}
	store, err := redishost.SecretStore()
	if err != nil {
		return "", Unavailable
	}
	encoded, err := store.Get(pendingNamespace, connection.PendingSecret)
	if err != nil {
		return "", Consumed
	}
	defer clear(encoded)
	var pending pendingPassword
	if json.Unmarshal(encoded, &pending) != nil || pending.DatabaseID != connection.DatabaseID || pending.CredentialID != connection.ID || pending.AccountID != connection.AccountID || pending.Generation != connection.Generation || pending.InstanceIdentity != instance.Identity || pending.InstanceGeneration == 0 || pending.InstanceGeneration > instance.Generation || pending.Scope != connection.Scope || pending.Subscribe != connection.Subscribe || pending.Publish != connection.Publish || pending.CreatedAt.IsZero() || pending.CreatedAt.After(time.Now().Add(time.Minute)) || len(pending.Password) != 64 || digestPassword(pending.Password) != connection.PasswordDigest {
		return "", Consumed
	}
	if err := retainPrimary(connection, pending.Password); err != nil {
		return "", err
	}
	return pending.Password, nil
}

func clearPrimary(connection *models.RedisConnection) error {
	if !connection.PrimaryConnection {
		return nil
	}
	store, err := redishost.SecretStore()
	if err != nil {
		return Unavailable
	}
	if err := store.Delete(primaryNamespace, primarySecretName(connection.ID)); err != nil {
		return Unavailable
	}
	return nil
}

func CurrentPrimary(ctx context.Context, name string) (*Issued, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, models.ScopeReadWrite, false, false, true, false)
	if err != nil {
		return nil, failure(err)
	}
	var result *Issued
	err = fenced(fresh, record.ID, func() error {
		fresh, current, err := authorize(fresh, name, models.ScopeReadWrite, false, false, true, false)
		if err != nil {
			return err
		}
		if current.ID != record.ID || current.ClusterID != record.ClusterID {
			return Conflict
		}
		var connection models.RedisConnection
		if database.DB.WithContext(fresh).Where("database_id = ? AND instance_identity = ? AND primary_connection = ?", record.ID, record.ClusterID, true).First(&connection).Error != nil {
			return Missing
		}
		if !connection.Usable(time.Now()) {
			return Conflict
		}
		instance, err := redishost.Instance(fresh, current)
		if err != nil {
			return Unavailable
		}
		valid, err := issuerAuthorized(fresh, current, &connection)
		if err != nil {
			return err
		}
		if !valid {
			return Conflict
		}
		password, err := recoverPrimary(&connection, instance)
		if err != nil {
			return err
		}
		if _, latest, err := authorize(fresh, name, models.ScopeReadWrite, false, false, true, false); err != nil {
			return err
		} else if latest.ID != record.ID || latest.ClusterID != record.ClusterID {
			return Conflict
		}
		result = issued(&connection, password)
		return nil
	})
	return result, failure(err)
}
