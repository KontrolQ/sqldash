package members

import (
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ShareDatabase(actor *access.AccountAccess, name string, recipientID uint, add bool) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		if actor == nil {
			return Refused
		}
		current, findError := access.LoadWith(transaction, actor.Account.ID)
		if findError != nil || !current.Account.Active {
			return Refused
		}
		var record models.Database
		if findError := access.Scope(transaction.Model(&models.Database{}), current).Where("name = ?", name).First(&record).Error; findError != nil {
			return Refused
		}
		if record.OwnerAccountID != current.Account.ID && !current.Can(permissions.DatabaseShare) {
			return Refused
		}
		var recipient models.Account
		if findError := transaction.First(&recipient, recipientID).Error; findError != nil || !recipient.Active || recipientID == record.OwnerAccountID {
			return Invalid
		}
		if add {
			return transaction.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.DatabaseShare{DatabaseID: record.ID, AccountID: recipientID, GrantedByID: current.Account.ID}).Error
		}
		return transaction.Where("database_id = ? AND account_id = ?", record.ID, recipientID).Delete(&models.DatabaseShare{}).Error
	})
}
