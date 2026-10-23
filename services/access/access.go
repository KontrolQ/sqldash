package access

import (
	"context"
	"gorm.io/gorm"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
)

type AccountAccess struct {
	Account models.Account
	Owner   bool
	Super   bool
	Rank    int
	Grants  map[string]bool
}

func Load(accountID uint) (*AccountAccess, error) { return LoadWith(database.DB, accountID) }

func LoadWith(store *gorm.DB, accountID uint) (*AccountAccess, error) {
	var account models.Account
	if findError := store.First(&account, accountID).Error; findError != nil {
		return nil, findError
	}
	result := &AccountAccess{Account: account, Grants: make(map[string]bool)}
	var site models.Site
	if findError := store.First(&site, 1).Error; findError != nil {
		return nil, findError
	}
	result.Owner = site.OwnerAccountID == accountID
	var roles []models.Role
	if findError := store.Joins("JOIN account_roles ON account_roles.role_id = roles.id").Where("account_roles.account_id = ?", accountID).Find(&roles).Error; findError != nil {
		return nil, findError
	}
	for _, role := range roles {
		if role.Position > result.Rank {
			result.Rank = role.Position
		}
		var grants []models.RolePermission
		if findError := store.Where("role_id = ?", role.ID).Find(&grants).Error; findError != nil {
			return nil, findError
		}
		for _, grant := range grants {
			result.Grants[grant.Permission] = true
		}
	}
	result.Super = result.Grants[permissions.Administrator]
	return result, nil
}

func (self *AccountAccess) Can(permission string) bool {
	return self != nil && self.Account.Active && permission != "" && (self.Owner || self.Super || permissions.Satisfied(self.Grants, permission))
}

func (self *AccountAccess) All() bool {
	return self != nil && self.Account.Active && (self.Owner || self.Super || self.Account.AllDatabases)
}

func (self *AccountAccess) CanManage(target *AccountAccess) bool {
	if self == nil || target == nil || !self.Can(permissions.UsersManage) || self.Account.ID == target.Account.ID || target.Owner {
		return false
	}
	if self.Owner {
		return true
	}
	return !target.Super && target.Rank < self.Rank
}

func (self *AccountAccess) CanGrant(role models.Role) bool {
	if !self.Can(permissions.RolesManage) && !self.Can(permissions.UsersManage) {
		return false
	}
	if self.Owner {
		return true
	}
	return role.Position < self.Rank
}

func Scope(query *gorm.DB, held *AccountAccess) *gorm.DB {
	if held == nil || !held.Account.Active {
		return query.Where("1 = 0")
	}
	if held.All() {
		return query
	}
	return query.Where("owner_account_id = ? OR id IN (SELECT database_id FROM database_shares WHERE account_id = ?)", held.Account.ID, held.Account.ID)
}

func Visible(held *AccountAccess) ([]models.Database, error) {
	return VisibleWith(database.DB, held)
}

func VisibleWith(store *gorm.DB, held *AccountAccess) ([]models.Database, error) {
	var records []models.Database
	findError := Scope(store.Model(&models.Database{}), held).Order("name asc").Find(&records).Error
	return records, findError
}

func Database(held *AccountAccess, name string) (*models.Database, error) {
	var record models.Database
	findError := Scope(database.DB.Model(&models.Database{}), held).Where("name = ?", name).First(&record).Error
	return &record, findError
}

func WithContext(parent context.Context, held *AccountAccess) context.Context {
	return context.WithValue(parent, contextKey{}, held)
}
func FromContext(parent context.Context) *AccountAccess {
	held, _ := parent.Value(contextKey{}).(*AccountAccess)
	return held
}
func AccountID(parent context.Context) uint {
	if held := FromContext(parent); held != nil {
		return held.Account.ID
	}
	return 0
}
