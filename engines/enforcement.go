package engines

import (
	"context"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"time"
)

func Quarantine(parent context.Context, record *models.Database) error {
	if record == nil {
		return DatabaseUnavailable
	}
	protected, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancel()
	updateError := database.DB.WithContext(protected).Model(&models.Database{}).Where("id = ?", record.ID).Updates(map[string]any{"state": models.DatabaseFailed, "policy_version": gorm.Expr("policy_version + 1")}).Error
	path := filepath.Join(config.PostgresRuntimePath(), "enforcement-blocked")
	file, createError := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if createError == nil {
		_, _ = file.WriteString(time.Now().UTC().Format(time.RFC3339))
		_ = file.Sync()
		_ = file.Close()
	}
	if updateError != nil {
		return updateError
	}
	if createError != nil && !os.IsExist(createError) {
		return createError
	}
	return nil
}
