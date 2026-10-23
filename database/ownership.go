package database

import (
	"errors"

	"sqldash/models"
	"sqldash/permissions"

	"gorm.io/gorm"
)

func migrateOwnership() error {
	return DB.Transaction(func(transaction *gorm.DB) error {
		var site models.Site
		findError := transaction.First(&site, 1).Error
		if errors.Is(findError, gorm.ErrRecordNotFound) {
			var first models.Account
			findError = transaction.Order("id asc").First(&first).Error
			if errors.Is(findError, gorm.ErrRecordNotFound) {
				return nil
			}
			if findError != nil {
				return findError
			}
			site = models.Site{ID: 1, OwnerAccountID: first.ID}
			if createError := transaction.Create(&site).Error; createError != nil {
				return createError
			}
			if updateError := transaction.Model(&first).Updates(map[string]any{"active": true, "all_databases": true}).Error; updateError != nil {
				return updateError
			}
		} else if findError != nil {
			return findError
		}
		if !site.LegacyRolesRemoved {
			if removeError := removeLegacyRoles(transaction); removeError != nil {
				return removeError
			}
			if updateError := transaction.Model(&site).Update("legacy_roles_removed", true).Error; updateError != nil {
				return updateError
			}
		}
		if updateError := transaction.Model(&models.Database{}).Where("owner_account_id = 0").Update("owner_account_id", site.OwnerAccountID).Error; updateError != nil {
			return updateError
		}
		return transaction.Model(&models.Token{}).Where("account_id = 0").Update("account_id", site.OwnerAccountID).Error
	})
}

func removeLegacyRoles(transaction *gorm.DB) error {
	var roles []models.Role
	if findError := transaction.Find(&roles).Error; findError != nil {
		return findError
	}
	for _, role := range roles {
		generated := role.Kind == permissions.SuperAdminKind
		var expected []string
		if role.Name == permissions.EditorName && role.Position == 20 && role.Color == "#3b82f6" {
			expected = permissions.LegacyEditor
		}
		if role.Name == permissions.ViewerName && role.Position == 10 && role.Color == "#22c55e" {
			expected = permissions.LegacyViewer
		}
		if expected != nil {
			var grants []models.RolePermission
			if findError := transaction.Where("role_id = ?", role.ID).Find(&grants).Error; findError != nil {
				return findError
			}
			if len(grants) == len(expected) {
				matching := make(map[string]bool)
				for _, key := range expected {
					matching[key] = true
				}
				generated = true
				for _, grant := range grants {
					if !matching[grant.Permission] {
						generated = false
					}
				}
			}
		}
		if !generated {
			continue
		}
		if deleteError := transaction.Where("role_id = ?", role.ID).Delete(&models.AccountRole{}).Error; deleteError != nil {
			return deleteError
		}
		if deleteError := transaction.Where("role_id = ?", role.ID).Delete(&models.RolePermission{}).Error; deleteError != nil {
			return deleteError
		}
		if deleteError := transaction.Unscoped().Delete(&role).Error; deleteError != nil {
			return deleteError
		}
	}
	return nil
}
