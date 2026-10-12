package account

import (
	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func ClearPicture(accountID uint) error {
	return setPicture(database.DB, accountID, "", "")
}

func DeletePicture(accountID uint, pictureID uint) error {
	deleted := database.DB.Unscoped().Where("id = ? AND account_id = ?", pictureID, accountID).Delete(&models.AccountPicture{})
	if deleted.Error != nil {
		return deleted.Error
	}
	if deleted.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
