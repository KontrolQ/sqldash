package members

import (
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"

	"gorm.io/gorm"
)

func ReorderRoles(actor *access.AccountAccess, identifiers []uint) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := currentActor(transaction, actor, permissions.RolesManage)
		if findError != nil {
			return findError
		}
		var roles []models.Role
		if findError := transaction.Find(&roles).Error; findError != nil {
			return findError
		}
		allowed := make(map[uint]bool)
		for _, role := range roles {
			if current.CanGrant(role) {
				allowed[role.ID] = true
			}
		}
		if len(identifiers) != len(allowed) || (!current.Owner && len(identifiers) >= current.Rank) {
			return Refused
		}
		seen := make(map[uint]bool)
		for _, identifier := range identifiers {
			if !allowed[identifier] || seen[identifier] {
				return Refused
			}
			seen[identifier] = true
		}
		for _, identifier := range identifiers {
			if updateError := transaction.Model(&models.Role{}).Where("id = ?", identifier).Update("position", -int(identifier)).Error; updateError != nil {
				return updateError
			}
		}
		seen = make(map[uint]bool)
		for index, identifier := range identifiers {
			if !allowed[identifier] || seen[identifier] {
				return Refused
			}
			seen[identifier] = true
			if updateError := transaction.Model(&models.Role{}).Where("id = ?", identifier).Update("position", len(identifiers)-index).Error; updateError != nil {
				return updateError
			}
		}
		return nil
	})
}
