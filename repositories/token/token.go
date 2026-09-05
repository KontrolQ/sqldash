package token

import (
	"errors"

	store "sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func ForDatabase(name string) ([]models.Token, error) {
	records := make([]models.Token, 0)

	if findError := store.DB.Where("database_name = ?", name).Order(NewestFirst).Find(&records).Error; findError != nil {
		return nil, findError
	}

	return records, nil
}

func FindByIdentifier(identifier string) (*models.Token, error) {
	held := &models.Token{}

	findError := store.DB.Where("identifier = ?", identifier).First(held).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if findError != nil {
		return nil, findError
	}

	return held, nil
}

func FindByDigest(digest string) (*models.Token, error) {
	held := &models.Token{}

	findError := store.DB.Where("digest = ?", digest).First(held).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if findError != nil {
		return nil, findError
	}

	return held, nil
}

func Create(record *models.Token) error {
	return store.DB.Create(record).Error
}

func Save(record *models.Token) error {
	return store.DB.Save(record).Error
}

func DeleteForDatabase(name string) error {
	return store.DB.Unscoped().Where("database_name = ?", name).Delete(&models.Token{}).Error
}
