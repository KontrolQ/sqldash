package database

import (
	"sqldash/models"
	"sqldash/utils/logger"
)

func migrate() {
	migrationError := DB.AutoMigrate(
		&models.Account{},
		&models.Session{},
		&models.Database{},
		&models.AccessRule{},
		&models.Token{},
		&models.Fingerprint{},
		&models.Statement{},
		&models.Rollup{},
	)

	if migrationError != nil {
		logger.Fatalf(LogPrefix, MigrationFailedLog, migrationError)
	}
}
