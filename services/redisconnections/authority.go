package redisconnections

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"

	"gorm.io/gorm"
)

func validateCreation(asked Creation) error {
	if strings.TrimSpace(asked.Label) != asked.Label || asked.Label == "" || len(asked.Label) > 120 || !utf8.ValidString(asked.Label) || (asked.Scope != models.ScopeReadOnly && asked.Scope != models.ScopeReadWrite) {
		return Invalid
	}
	for _, character := range asked.Label {
		if unicode.IsControl(character) {
			return Invalid
		}
	}
	if asked.ExpiresAt != nil && (!asked.ExpiresAt.After(time.Now()) || asked.ExpiresAt.Year() > 9999) {
		return Invalid
	}
	return nil
}

func allowed(actor *access.AccountAccess, scope models.TokenScope, subscribe, publish, primary bool) bool {
	return actor != nil && actor.Can(permissions.RedisCredentialsManage) && actor.Can(permissions.RedisDataRead) && (scope == models.ScopeReadOnly || scope == models.ScopeReadWrite) && (scope != models.ScopeReadWrite || actor.Can(permissions.RedisDataWrite)) && (!primary || scope == models.ScopeReadWrite && actor.Can(permissions.RedisDataWrite)) && (!subscribe || actor.Can(permissions.RedisSubscribe)) && (!publish || actor.Can(permissions.RedisPublish))
}

func authorize(ctx context.Context, name string, scope models.TokenScope, subscribe, publish, primary, applying bool) (context.Context, *models.Database, error) {
	fresh, actor, err := access.Refresh(ctx)
	if err != nil || !allowed(actor, scope, subscribe, publish, primary) {
		return ctx, nil, Refused
	}
	var record models.Database
	err = access.Scope(database.DB.WithContext(fresh).Model(&models.Database{}), actor).Where("name = ?", name).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fresh, nil, Missing
	}
	if err != nil {
		return fresh, nil, Unavailable
	}
	if record.Kind() != models.EngineRedis {
		return fresh, nil, Missing
	}
	if record.ClusterID == "" || record.State != models.DatabaseReady && (!applying || record.State != models.DatabaseApplying) {
		return fresh, nil, Conflict
	}
	return fresh, &record, nil
}

func issuerAuthorized(ctx context.Context, record *models.Database, connection *models.RedisConnection) (bool, error) {
	store := database.DB.WithContext(ctx)
	actor, err := access.LoadWith(store, connection.AccountID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, Unavailable
	}
	if !allowed(actor, connection.Scope, connection.Subscribe, connection.Publish, connection.PrimaryConnection) {
		return false, nil
	}
	var count int64
	if err := access.Scope(store.Model(&models.Database{}), actor).Where("id = ? AND engine = ? AND cluster_id = ? AND state IN ?", record.ID, models.EngineRedis, connection.InstanceIdentity, []string{models.DatabaseReady, models.DatabaseApplying}).Count(&count).Error; err != nil {
		return false, Unavailable
	}
	return count == 1, nil
}

func loadConnection(ctx context.Context, databaseID, identifier uint) (*models.RedisConnection, error) {
	if identifier == 0 {
		return nil, Missing
	}
	var connection models.RedisConnection
	if err := database.DB.WithContext(ctx).Where("id = ? AND database_id = ?", identifier, databaseID).First(&connection).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, Missing
		}
		return nil, Unavailable
	}
	return &connection, nil
}
