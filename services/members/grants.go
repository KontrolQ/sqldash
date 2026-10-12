package members

import (
	"encoding/json"
	"sqldash/database"
	"sqldash/models"
	"sqldash/services/access"

	"gorm.io/gorm"
)

func ValidateGrants(actor *access.AccountAccess, roles []uint, databases []uint, all bool) error {
	return validateGrantsWith(database.DB, actor, roles, databases, all)
}

func validateGrantsWith(store *gorm.DB, actor *access.AccountAccess, roles []uint, databases []uint, all bool) error {
	if actor == nil || !actor.Account.Active || (all && !actor.All()) {
		return Refused
	}
	for _, identifier := range roles {
		var role models.Role
		if findError := store.First(&role, identifier).Error; findError != nil || !actor.CanGrant(role) {
			return Refused
		}
		var grants []models.RolePermission
		if findError := store.Where("role_id = ?", role.ID).Find(&grants).Error; findError != nil {
			return findError
		}
		for _, grant := range grants {
			if !actor.Can(grant.Permission) {
				return Refused
			}
		}
	}
	visible, findError := access.VisibleWith(store, actor)
	if findError != nil {
		return findError
	}
	allowed := make(map[uint]bool)
	for _, record := range visible {
		allowed[record.ID] = true
	}
	for _, identifier := range databases {
		if !allowed[identifier] {
			return Refused
		}
	}
	return nil
}

func Assign(transaction *gorm.DB, accountID uint, roles []uint, databases []uint, actorID uint) error {
	if deleteError := transaction.Where("account_id = ?", accountID).Delete(&models.AccountRole{}).Error; deleteError != nil {
		return deleteError
	}
	if deleteError := transaction.Where("account_id = ?", accountID).Delete(&models.DatabaseShare{}).Error; deleteError != nil {
		return deleteError
	}
	roleSet := make(map[uint]bool)
	for _, identifier := range roles {
		if roleSet[identifier] {
			continue
		}
		roleSet[identifier] = true
		if createError := transaction.Create(&models.AccountRole{AccountID: accountID, RoleID: identifier}).Error; createError != nil {
			return createError
		}
	}
	databaseSet := make(map[uint]bool)
	for _, identifier := range databases {
		if databaseSet[identifier] {
			continue
		}
		databaseSet[identifier] = true
		if createError := transaction.Create(&models.DatabaseShare{DatabaseID: identifier, AccountID: accountID, GrantedByID: actorID}).Error; createError != nil {
			return createError
		}
	}
	return nil
}

func encodeIDs(values []uint) string { encoded, _ := json.Marshal(values); return string(encoded) }
func decodeIDs(encoded string) []uint {
	var values []uint
	json.Unmarshal([]byte(encoded), &values)
	return values
}
