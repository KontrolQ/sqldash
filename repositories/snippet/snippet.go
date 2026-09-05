package snippet

import (
	"errors"

	store "sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func ForDatabase(databaseName string) ([]models.Snippet, error) {
	records := make([]models.Snippet, 0)

	findError := store.DB.Where("database = ?", databaseName).Order(NameOrder).Find(&records).Error
	if findError != nil {
		return nil, findError
	}

	return records, nil
}

func FindByID(identifier uint) (*models.Snippet, error) {
	held := &models.Snippet{}

	findError := store.DB.First(held, identifier).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if findError != nil {
		return nil, findError
	}

	return held, nil
}

func Create(record *models.Snippet) error {
	return store.DB.Create(record).Error
}

func Save(record *models.Snippet) error {
	return store.DB.Save(record).Error
}

func Delete(identifier uint) error {
	return store.DB.Delete(&models.Snippet{}, identifier).Error
}
