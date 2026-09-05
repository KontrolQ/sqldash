package account

import (
	"errors"

	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func Count() (int64, error) {
	var total int64

	if countError := database.DB.Model(&models.Account{}).Count(&total).Error; countError != nil {
		return 0, countError
	}

	return total, nil
}

func FindByID(accountID uint) (*models.Account, error) {
	held := &models.Account{}

	findError := database.DB.First(held, accountID).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if findError != nil {
		return nil, findError
	}

	return held, nil
}

func FindByUsername(username string) (*models.Account, error) {
	held := &models.Account{}

	findError := database.DB.Where("username = ?", username).First(held).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if findError != nil {
		return nil, findError
	}

	return held, nil
}

func Create(record *models.Account) error {
	return database.DB.Create(record).Error
}
