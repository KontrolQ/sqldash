package telemetry

import (
	"gorm.io/gorm"
	"sqldash/database"
	"sqldash/models"
)

func DeleteForDatabase(name string) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		for _, record := range []any{&models.Statement{}, &models.Fingerprint{}, &models.Rollup{}} {
			if deleteError := transaction.Unscoped().Where("database_name = ?", name).Delete(record).Error; deleteError != nil {
				return deleteError
			}
		}
		return nil
	})
}
