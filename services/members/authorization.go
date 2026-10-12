package members

import (
	"sqldash/permissions"
	"sqldash/services/access"

	"gorm.io/gorm"
)

func currentActor(transaction *gorm.DB, actor *access.AccountAccess, permission string) (*access.AccountAccess, error) {
	if actor == nil {
		return nil, Refused
	}
	current, findError := access.LoadWith(transaction, actor.Account.ID)
	if findError != nil {
		return nil, findError
	}
	if !current.Can(permission) {
		return nil, Refused
	}
	return current, nil
}

func managedAccount(transaction *gorm.DB, actor *access.AccountAccess, accountID uint) (*access.AccountAccess, error) {
	current, findError := currentActor(transaction, actor, permissions.UsersManage)
	if findError != nil {
		return nil, findError
	}
	target, findError := access.LoadWith(transaction, accountID)
	if findError != nil {
		return nil, findError
	}
	if !current.CanManage(target) {
		return nil, Refused
	}
	return current, nil
}
