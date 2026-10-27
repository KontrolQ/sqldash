package account

import (
	"context"

	"sqldash/database"
	"sqldash/models"
)

func PictureVisible(ctx context.Context, accountID uint, reference string, mediaType string) (bool, error) {
	var current int64
	if readingError := database.DB.WithContext(ctx).Model(&models.Account{}).Where("profile_picture = ? AND profile_picture_type = ?", reference, mediaType).Count(&current).Error; readingError != nil {
		return false, readingError
	}
	if current > 0 {
		return true, nil
	}
	var historical int64
	if readingError := database.DB.WithContext(ctx).Model(&models.AccountPicture{}).Where("account_id = ? AND profile_picture = ? AND profile_picture_type = ?", accountID, reference, mediaType).Count(&historical).Error; readingError != nil {
		return false, readingError
	}
	return historical > 0, nil
}
