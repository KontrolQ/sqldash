package database

import (
	"sqldash/models"
	"sqldash/utils/logger"
)

func migrate() {
	migrationError := DB.AutoMigrate(
		&models.Account{},
		&models.AccountPicture{},
		&models.Site{}, &models.Role{}, &models.RolePermission{}, &models.AccountRole{},
		&models.DatabaseShare{}, &models.AccountLink{}, &models.OwnershipTransfer{}, &models.SMTPSettings{},
		&models.Session{},
		&models.Database{},
		&models.AccessRule{},
		&models.Token{},
		&models.Fingerprint{},
		&models.Statement{},
		&models.Rollup{},
		&models.Marker{},
		&models.Snippet{},
		&models.Event{},
	)

	if migrationError != nil {
		logger.Fatalf(LogPrefix, MigrationFailedLog, migrationError)
	}
	if ownershipError := migrateOwnership(); ownershipError != nil {
		logger.Fatalf(LogPrefix, MigrationFailedLog, ownershipError)
	}
}
