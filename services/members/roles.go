package members

import (
	"regexp"
	"strings"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"

	"gorm.io/gorm"
)

var roleColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func SaveRole(actor *access.AccountAccess, identifier uint, name string, color string, position int, grants []string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 || !roleColor.MatchString(color) || position < 0 || position >= permissions.SuperAdminPosition {
		return Invalid
	}
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := currentActor(transaction, actor, permissions.RolesManage)
		if findError != nil {
			return findError
		}
		role := models.Role{}
		if identifier != 0 {
			if findError := transaction.First(&role, identifier).Error; findError != nil {
				return findError
			}
			if !current.CanGrant(role) {
				return Refused
			}
		}
		if position == 0 {
			if identifier != 0 {
				position = role.Position
			} else {
				if shiftError := makeRoomForRole(transaction, current); shiftError != nil {
					return shiftError
				}
				position = 1
			}
		}
		if !current.Owner && position >= current.Rank {
			return Refused
		}
		var occupied int64
		if findError := transaction.Model(&models.Role{}).Where("position = ? AND id <> ?", position, identifier).Count(&occupied).Error; findError != nil {
			return findError
		}
		if occupied != 0 {
			return Invalid
		}
		allowed := make(map[string]bool)
		for _, key := range grants {
			if !permissions.Known(key) || !current.Can(key) {
				return Refused
			}
			allowed[key] = true
		}
		if !allowed[permissions.Administrator] {
			for key := range allowed {
				if !permissions.Satisfied(allowed, key) {
					return DependenciesInvalid
				}
			}
		}
		role.Name = name
		role.Color = color
		role.Position = position
		if saveError := transaction.Save(&role).Error; saveError != nil {
			return saveError
		}
		if deleteError := transaction.Where("role_id = ?", role.ID).Delete(&models.RolePermission{}).Error; deleteError != nil {
			return deleteError
		}
		for key := range allowed {
			if createError := transaction.Create(&models.RolePermission{RoleID: role.ID, Permission: key}).Error; createError != nil {
				return createError
			}
		}
		return nil
	})
}

func makeRoomForRole(transaction *gorm.DB, actor *access.AccountAccess) error {
	var roles []models.Role
	if findError := transaction.Order("position desc, id asc").Find(&roles).Error; findError != nil {
		return findError
	}
	var movable []models.Role
	for _, role := range roles {
		if actor.CanGrant(role) {
			movable = append(movable, role)
		}
	}
	if len(movable)+1 >= permissions.SuperAdminPosition || (!actor.Owner && len(movable)+1 >= actor.Rank) {
		return Refused
	}
	for index, role := range movable {
		if updateError := transaction.Model(&models.Role{}).Where("id = ?", role.ID).Update("position", len(movable)+1-index).Error; updateError != nil {
			return updateError
		}
	}
	return nil
}

func DeleteRole(actor *access.AccountAccess, identifier uint) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := currentActor(transaction, actor, permissions.RolesManage)
		if findError != nil {
			return findError
		}
		var role models.Role
		if findError := transaction.First(&role, identifier).Error; findError != nil {
			return findError
		}
		if !current.CanGrant(role) {
			return Refused
		}
		if deleteError := transaction.Where("role_id = ?", role.ID).Delete(&models.AccountRole{}).Error; deleteError != nil {
			return deleteError
		}
		if deleteError := transaction.Where("role_id = ?", role.ID).Delete(&models.RolePermission{}).Error; deleteError != nil {
			return deleteError
		}
		return transaction.Unscoped().Delete(&role).Error
	})
}
