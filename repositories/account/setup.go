package account

import (
	"errors"

	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func CreateFirst(record *models.Account) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		var total int64
		if countError := transaction.Model(&models.Account{}).Count(&total).Error; countError != nil {
			return countError
		}
		if total != 0 {
			return errors.New("This instance is already set up.")
		}
		record.Active = true
		record.AllDatabases = true
		if createError := transaction.Create(record).Error; createError != nil {
			return createError
		}
		return transaction.Create(&models.Site{ID: 1, OwnerAccountID: record.ID}).Error
	})
}
