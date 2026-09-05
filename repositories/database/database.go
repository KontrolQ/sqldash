package database

import (
	"errors"

	store "sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func All() ([]models.Database, error) {
	records := make([]models.Database, 0)

	if findError := store.DB.Order(NameOrder).Find(&records).Error; findError != nil {
		return nil, findError
	}

	return records, nil
}

func FindByName(name string) (*models.Database, error) {
	held := &models.Database{}

	findError := store.DB.Where("name = ?", name).First(held).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if findError != nil {
		return nil, findError
	}

	return held, nil
}

func Create(record *models.Database) error {
	return store.DB.Create(record).Error
}

func Delete(record *models.Database) error {
	return store.DB.Unscoped().Delete(record).Error
}
