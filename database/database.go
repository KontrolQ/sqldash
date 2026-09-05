package database

import (
	"sqldash/config"
	"sqldash/utils/logger"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	var connectionError error

	DB, connectionError = gorm.Open(sqlite.Open(config.StatePath()), &gorm.Config{
		Logger: resolveGORMLogLevel(),
	})

	if connectionError != nil {
		logger.Fatalf(LogPrefix, ConnectionFailedLog, connectionError)
	}

	handle, poolError := DB.DB()
	if poolError != nil {
		logger.Fatalf(LogPrefix, PoolConfigFailedLog, poolError)
	}

	handle.SetMaxOpenConns(MaxOpenConnections)
	handle.SetMaxIdleConns(MaxIdleConnections)
	handle.SetConnMaxLifetime(MaxConnectionLifetime)

	for _, pragma := range StartupPragmas {
		if pragmaError := DB.Exec(pragma).Error; pragmaError != nil {
			logger.Fatalf(LogPrefix, PragmaFailedLog, pragma, pragmaError)
		}
	}

	logger.Successf(LogPrefix, ConnectedLog, config.StatePath())

	migrate()
}
