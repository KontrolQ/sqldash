package settings

import (
	"errors"
	"net/http"

	repository "sqldash/repositories/account"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func ClearPicture(accountID uint) *fiber.Error {
	if clearError := repository.ClearPicture(accountID); clearError != nil {
		logger.Errorf(LogPrefix, SaveFailedLog, clearError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}
	return nil
}

func DeletePicture(accountID uint, pictureID uint) *fiber.Error {
	if deleteError := repository.DeletePicture(accountID, pictureID); deleteError != nil {
		if errors.Is(deleteError, gorm.ErrRecordNotFound) {
			return shortcuts.ServiceError(http.StatusNotFound, PictureUnavailable)
		}
		logger.Errorf(LogPrefix, SaveFailedLog, deleteError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}
	return nil
}
