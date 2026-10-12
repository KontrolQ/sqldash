package members

import (
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ChangeRoleMembers(actor *access.AccountAccess, roleID uint, accountIDs []uint, add bool) error {
	if len(accountIDs) == 0 || len(accountIDs) > 100 {
		return Invalid
	}
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := currentActor(transaction, actor, permissions.RolesManage)
		if findError != nil || !current.Can(permissions.UsersManage) {
			return Refused
		}
		if grantError := validateGrantsWith(transaction, current, []uint{roleID}, nil, false); grantError != nil {
			return grantError
		}
		seen := make(map[uint]bool)
		for _, accountID := range accountIDs {
			if seen[accountID] {
				continue
			}
			seen[accountID] = true
			target, findError := access.LoadWith(transaction, accountID)
			if findError != nil || !current.CanManage(target) || (add && !target.Account.Active) {
				return Refused
			}
			if add {
				if createError := transaction.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.AccountRole{AccountID: accountID, RoleID: roleID}).Error; createError != nil {
					return createError
				}
			} else {
				if deleteError := transaction.Where("account_id = ? AND role_id = ?", accountID, roleID).Delete(&models.AccountRole{}).Error; deleteError != nil {
					return deleteError
				}
			}
		}
		return nil
	})
}

func RoleMemberChoices(actor *access.AccountAccess, roleID uint) ([]UserView, []UserView, error) {
	var accounts []models.Account
	if findError := database.DB.Order("username asc").Find(&accounts).Error; findError != nil {
		return nil, nil, findError
	}
	var assigned []models.AccountRole
	if findError := database.DB.Where("role_id = ?", roleID).Find(&assigned).Error; findError != nil {
		return nil, nil, findError
	}
	assignedIDs := make(map[uint]bool)
	for _, assignment := range assigned {
		assignedIDs[assignment.AccountID] = true
	}
	var held, available []UserView
	for _, account := range accounts {
		target, findError := access.Load(account.ID)
		if findError != nil {
			return nil, nil, findError
		}
		view, viewError := userView(target)
		if viewError != nil {
			return nil, nil, viewError
		}
		view.Manage = actor.CanManage(target)
		if assignedIDs[account.ID] {
			held = append(held, *view)
		} else if view.Manage && account.Active {
			available = append(available, *view)
		}
	}
	return held, available, nil
}
