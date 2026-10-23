package database

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sqldash/models"
	"sqldash/permissions"
	"time"
)

func migrateEngines() error {
	return DB.Transaction(func(transaction *gorm.DB) error {
		if updateError := transaction.Model(&models.Database{}).Where("engine IS NULL OR engine = ''").Update("engine", models.EngineLibSQL).Error; updateError != nil {
			return updateError
		}
		if updateError := transaction.Model(&models.Database{}).Where("state IS NULL OR state = ''").Update("state", models.DatabaseReady).Error; updateError != nil {
			return updateError
		}
		var marker models.SchemaMigration
		findError := transaction.First(&marker, "name = ?", "database-permission-namespaces").Error
		if findError == nil {
			return nil
		}
		if !errors.Is(findError, gorm.ErrRecordNotFound) {
			return findError
		}
		if updateError := transaction.Model(&models.Event{}).Where("database_id = 0 AND database <> ''").Updates(map[string]any{
			"database_id": gorm.Expr("COALESCE((SELECT id FROM databases WHERE databases.name = events.database AND databases.deleted_at IS NULL), 0)"),
			"engine":      gorm.Expr("COALESCE((SELECT engine FROM databases WHERE databases.name = events.database AND databases.deleted_at IS NULL), 'libsql')"),
		}).Error; updateError != nil {
			return updateError
		}
		for _, target := range []any{&models.Statement{}, &models.Fingerprint{}, &models.Rollup{}} {
			if updateError := transaction.Model(target).Where("database_id = 0").Update("database_id", gorm.Expr("COALESCE((SELECT id FROM databases WHERE databases.name = database_name AND databases.deleted_at IS NULL), 0)")).Error; updateError != nil {
				return updateError
			}
		}
		for previous, next := range permissions.LegacyPermissions {
			var grants []models.RolePermission
			if findError := transaction.Where("permission = ?", previous).Find(&grants).Error; findError != nil {
				return findError
			}
			for _, grant := range grants {
				converted := models.RolePermission{RoleID: grant.RoleID, Permission: next}
				if createError := transaction.Clauses(clause.OnConflict{DoNothing: true}).Create(&converted).Error; createError != nil {
					return createError
				}
			}
			if previous != permissions.PlatformAuditView {
				if deleteError := transaction.Where("permission = ?", previous).Delete(&models.RolePermission{}).Error; deleteError != nil {
					return deleteError
				}
			}
		}
		return transaction.Create(&models.SchemaMigration{Name: "database-permission-namespaces", AppliedAt: time.Now().UTC()}).Error
	})
}
