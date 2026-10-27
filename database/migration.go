package database

import (
	"context"

	"sqldash/models"
)

func migrate(ctx context.Context) error {
	if migrationError := migrateOIDColumns(); migrationError != nil {
		return migrationError
	}
	if migrationError := DB.AutoMigrate(
		&models.Account{},
		&models.AccountPicture{},
		&models.Site{}, &models.Role{}, &models.RolePermission{}, &models.AccountRole{},
		&models.DatabaseShare{}, &models.AccountLink{}, &models.OwnershipTransfer{}, &models.SMTPSettings{},
		&models.Session{}, &models.Database{}, &models.AccessRule{}, &models.Token{},
		&models.Fingerprint{}, &models.Statement{}, &models.Rollup{}, &models.Marker{},
		&models.Snippet{}, &models.Event{}, &models.PostgresConnection{}, &models.DatabaseOperation{},
		&models.SchemaMigration{}, &models.PostgresStatistics{}, &models.PostgresStatementStatistics{},
		&models.RedisInstance{}, &models.RedisConnection{}, &models.RedisStatistics{},
	); migrationError != nil {
		return migrationError
	}
	if ownershipError := migrateOwnership(); ownershipError != nil {
		return ownershipError
	}
	if migrationError := migrateEngines(); migrationError != nil {
		return migrationError
	}
	return migratePicturesWith(ctx, DB)
}
