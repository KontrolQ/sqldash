package redisconnections

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/redishost"
	"sqldash/services/access"
)

func randomValue(length int) (string, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return "", Unavailable
	}
	encoded := hex.EncodeToString(value)
	clear(value)
	return encoded, nil
}

func digestPassword(password string) string {
	digest := sha256.Sum256([]byte(password))
	return hex.EncodeToString(digest[:])
}

func stagePassword(ctx context.Context, record *models.Database, connection *models.RedisConnection, instance *models.RedisInstance) error {
	store, err := redishost.SecretStore()
	if err != nil {
		return Unavailable
	}
	password, err := randomValue(32)
	if err != nil {
		return err
	}
	reference, err := randomValue(24)
	if err != nil {
		return err
	}
	previousState := connection.State
	connection.PasswordDigest = digestPassword(password)
	connection.PendingSecret = reference
	connection.State = models.DatabasePending
	if connection.ID == 0 {
		if err := database.DB.WithContext(ctx).Create(connection).Error; err != nil {
			return Conflict
		}
	} else {
		previous := connection.Generation
		if previous == 0 || previous >= 1<<63-1 {
			return Conflict
		}
		connection.Generation++
		updated := database.DB.WithContext(ctx).Model(&models.RedisConnection{}).Where("id = ? AND database_id = ? AND instance_identity = ? AND generation = ? AND state = ?", connection.ID, record.ID, instance.Identity, previous, previousState).Updates(map[string]any{"generation": connection.Generation, "account_id": connection.AccountID, "password_digest": connection.PasswordDigest, "pending_secret": reference, "state": models.DatabasePending, "revoked_at": nil})
		if updated.Error != nil || updated.RowsAffected != 1 {
			return Conflict
		}
		connection.RevokedAt = nil
	}
	pending := pendingPassword{DatabaseID: record.ID, CredentialID: connection.ID, AccountID: connection.AccountID, Generation: connection.Generation, InstanceIdentity: instance.Identity, InstanceGeneration: instance.Generation, Scope: connection.Scope, Subscribe: connection.Subscribe, Publish: connection.Publish, CreatedAt: time.Now().UTC(), Password: password}
	encoded, err := json.Marshal(pending)
	if err != nil {
		return Unavailable
	}
	defer clear(encoded)
	if err := store.Set(pendingNamespace, reference, encoded); err != nil {
		if removeError := redishost.RemoveUser(ctx, record, connection.Username); removeError != nil {
			return quarantine(ctx, record, instance)
		}
		return Unavailable
	}
	return retainPrimary(connection, password)
}

func loadPending(connection *models.RedisConnection, instance *models.RedisInstance) (*pendingPassword, error) {
	if connection.PendingSecret == "" {
		return nil, Consumed
	}
	store, err := redishost.SecretStore()
	if err != nil {
		return nil, Unavailable
	}
	encoded, err := store.Get(pendingNamespace, connection.PendingSecret)
	if err != nil {
		return nil, Consumed
	}
	defer clear(encoded)
	var pending pendingPassword
	if json.Unmarshal(encoded, &pending) != nil || pending.DatabaseID != connection.DatabaseID || pending.CredentialID != connection.ID || pending.AccountID != connection.AccountID || pending.Generation != connection.Generation || pending.InstanceIdentity != connection.InstanceIdentity || pending.InstanceIdentity != instance.Identity || pending.InstanceGeneration != instance.Generation || pending.Scope != connection.Scope || pending.Subscribe != connection.Subscribe || pending.Publish != connection.Publish || pending.CreatedAt.IsZero() || pending.CreatedAt.After(time.Now().Add(time.Minute)) || time.Since(pending.CreatedAt) > pendingLifetime || len(pending.Password) != 64 || digestPassword(pending.Password) != connection.PasswordDigest {
		return nil, Consumed
	}
	return &pending, nil
}

func clearPending(ctx context.Context, connection *models.RedisConnection, state string) error {
	if connection.PendingSecret == "" {
		return nil
	}
	store, err := redishost.SecretStore()
	if err != nil {
		return Unavailable
	}
	if err := store.Delete(pendingNamespace, connection.PendingSecret); err != nil {
		return Unavailable
	}
	updated := database.DB.WithContext(ctx).Unscoped().Model(&models.RedisConnection{}).Where("id = ? AND generation = ? AND state = ? AND pending_secret = ?", connection.ID, connection.Generation, connection.State, connection.PendingSecret).Updates(map[string]any{"pending_secret": "", "state": state})
	if updated.Error != nil || updated.RowsAffected != 1 {
		return Conflict
	}
	connection.PendingSecret = ""
	connection.State = state
	return nil
}

func deliver(ctx context.Context, record *models.Database, connection *models.RedisConnection, instance *models.RedisInstance) (*Issued, error) {
	if connection.AccountID != access.AccountID(ctx) || !connection.Usable(time.Now()) {
		return nil, Refused
	}
	pending, err := loadPending(connection, instance)
	if err != nil {
		return nil, err
	}
	updated := database.DB.WithContext(ctx).Model(&models.RedisConnection{}).Where("id = ? AND database_id = ? AND account_id = ? AND generation = ? AND state = ? AND pending_secret = ?", connection.ID, record.ID, access.AccountID(ctx), connection.Generation, models.DatabaseReady, connection.PendingSecret).Update("state", stateDelivering)
	if updated.Error != nil || updated.RowsAffected != 1 {
		return nil, Conflict
	}
	connection.State = stateDelivering
	if err := clearPending(ctx, connection, models.DatabaseReady); err != nil {
		return nil, err
	}
	fresh, current, err := authorize(ctx, record.Name, connection.Scope, connection.Subscribe, connection.Publish, connection.PrimaryConnection, false)
	if err != nil || current.ID != record.ID || current.ClusterID != connection.InstanceIdentity {
		return nil, Refused
	}
	instanceNow, err := redishost.Instance(fresh, current)
	if err != nil || instanceNow.Generation != instance.Generation {
		return nil, Conflict
	}
	return issued(connection, pending.Password), nil
}

func issued(connection *models.RedisConnection, password string) *Issued {
	address := url.URL{Scheme: "rediss", Host: net.JoinHostPort(config.RedisHost(), strconv.Itoa(config.Redis.PublicPort)), Path: "/0", User: url.UserPassword(connection.Username, password)}
	return &Issued{CredentialID: connection.ID, Username: connection.Username, Password: password, URL: address.String(), Scope: connection.Scope, ExpiresAt: connection.ExpiresAt}
}
