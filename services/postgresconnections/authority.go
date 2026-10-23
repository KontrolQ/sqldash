package postgresconnections

import (
	"context"
	"errors"
	"time"

	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"

	"gorm.io/gorm"
)

func credentialAuthorityWith(ctx context.Context, store *gorm.DB, record *models.Database, operation *models.DatabaseOperation, generation uint64) (context.Context, *models.PostgresConnection, error) {
	actor, err := access.LoadWith(store, operation.AccountID)
	if err != nil || actor == nil || !actor.Account.Active || !actor.Can(permissions.PostgresCredentialsManage) || !actor.Can(permissions.PostgresDataRead) {
		return ctx, nil, errors.New(AuthorityRefused)
	}
	var current models.Database
	if err := access.Scope(store.Model(&models.Database{}), actor).First(&current, operation.DatabaseID).Error; err != nil {
		return ctx, nil, errors.New(AuthorityRefused)
	}
	if current.ID != record.ID || current.Kind() != models.EnginePostgres || operation.Engine != models.EnginePostgres || !current.Available() {
		return ctx, nil, errors.New(AuthorityRefused)
	}
	var connection models.PostgresConnection
	if err := store.Where("id = ? AND database_id = ? AND generation = ?", operation.ConnectionID, current.ID, generation).First(&connection).Error; err != nil {
		return ctx, nil, errors.New(AuthorityRefused)
	}
	if operation.Kind != "credential-revoke" {
		if connection.AccountID != operation.AccountID || connection.RevokedAt != nil || connection.ExpiresAt != nil && !connection.ExpiresAt.After(time.Now()) || connection.Scope == models.ScopeReadWrite && !actor.Can(permissions.PostgresDataWrite) {
			return ctx, nil, errors.New(AuthorityRefused)
		}
	}
	return access.WithContext(ctx, actor), &connection, nil
}
