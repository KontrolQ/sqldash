package access

import (
	"gorm.io/gorm"
	"sqldash/models"
	"sqldash/permissions"
)

func (self *AccountAccess) CanDatabase(record *models.Database, permission string) bool {
	return record != nil && self.Can(permissions.ForEngine(permission, record.Kind()))
}

func (self *AccountAccess) CanAny(permission string) bool {
	return self.Can(permission) || self.Can(permissions.ForEngine(permission, models.EnginePostgres)) || self.Can(permissions.ForEngine(permission, models.EngineRedis))
}

func (self *AccountAccess) CanShare(record *models.Database) bool {
	return self != nil && self.Account.Active && record != nil && (record.OwnerAccountID == self.Account.ID || self.CanDatabase(record, permissions.DatabaseShare))
}

func VisibleFor(held *AccountAccess, permission string) ([]models.Database, error) {
	records, findError := Visible(held)
	if findError != nil {
		return nil, findError
	}
	return withPermission(records, held, permission), nil
}

func VisibleForWith(store *gorm.DB, held *AccountAccess, permission string) ([]models.Database, error) {
	records, findError := VisibleWith(store, held)
	if findError != nil {
		return nil, findError
	}
	return withPermission(records, held, permission), nil
}

func withPermission(records []models.Database, held *AccountAccess, permission string) []models.Database {
	visible := make([]models.Database, 0, len(records))
	for _, record := range records {
		if held.CanDatabase(&record, permission) {
			visible = append(visible, record)
		}
	}
	return visible
}
