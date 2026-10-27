package database

import (
	"context"
	"os"
	"sqldash/config"
	"sqldash/models"
	"sync"
)

var initializationGuard sync.Mutex

func Initialize(parent context.Context) error {
	initializationGuard.Lock()
	defer initializationGuard.Unlock()
	file, openError := os.OpenFile(config.StatePath()+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if openError != nil {
		return openError
	}
	defer file.Close()
	if lockError := lockSchema(parent, file); lockError != nil {
		return lockError
	}
	defer unlockSchema(file)
	return migrate(parent)
}

func Verify() error {
	var marker models.SchemaMigration
	return DB.First(&marker, "name = ?", SchemaReadyMarker).Error
}
