package account

import (
	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/utils/collections"

	"gorm.io/gorm"
)

func setPicture(transaction *gorm.DB, accountID uint, encoded string, mediaType string) error {
	updated := transaction.Model(&models.Account{}).Where("id = ?", accountID).Updates(collections.Record[string, any]{
		"profile_picture":      encoded,
		"profile_picture_type": mediaType,
	})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func addPicture(transaction *gorm.DB, accountID uint, encoded string, mediaType string) error {
	return transaction.Create(&models.AccountPicture{AccountID: accountID, ProfilePicture: encoded, ProfilePictureType: mediaType}).Error
}

func SavePicture(accountID uint, encoded string, mediaType string) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		var account models.Account
		if findError := transaction.First(&account, accountID).Error; findError != nil {
			return findError
		}
		if account.ProfilePicture == encoded && account.ProfilePictureType == mediaType {
			return nil
		}

		if saveError := addPicture(transaction, accountID, encoded, mediaType); saveError != nil {
			return saveError
		}
		if saveError := setPicture(transaction, accountID, encoded, mediaType); saveError != nil {
			return saveError
		}

		if config.History.MaximumPictures == 0 {
			return nil
		}
		retained := transaction.Model(&models.AccountPicture{}).Select("id").Where("account_id = ?", accountID).Order("id desc").Limit(config.History.MaximumPictures)
		return transaction.Unscoped().Where("account_id = ? AND id NOT IN (?)", accountID, retained).Delete(&models.AccountPicture{}).Error
	})
}

func PictureHistory(accountID uint) ([]models.AccountPicture, error) {
	var history []models.AccountPicture
	query := database.DB.Where("account_id = ?", accountID).Order("id desc")
	if config.History.MaximumPictures > 0 {
		query = query.Limit(config.History.MaximumPictures)
	}
	findError := query.Find(&history).Error
	return history, findError
}

func RestorePicture(accountID uint, pictureID uint) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		var picture models.AccountPicture
		if findError := transaction.Where("id = ? AND account_id = ?", pictureID, accountID).First(&picture).Error; findError != nil {
			return findError
		}
		return setPicture(transaction, accountID, picture.ProfilePicture, picture.ProfilePictureType)
	})
}
