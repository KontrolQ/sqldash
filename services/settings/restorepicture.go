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

func RestorePicture(accountID uint, pictureID uint) *fiber.Error {
	if restoreError := repository.RestorePicture(accountID, pictureID); restoreError != nil {
		if errors.Is(restoreError, gorm.ErrRecordNotFound) {
			return shortcuts.ServiceError(http.StatusNotFound, PictureUnavailable)
		}
		logger.Errorf(LogPrefix, SaveFailedLog, restoreError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}
	return nil
}
