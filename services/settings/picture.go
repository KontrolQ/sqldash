package settings

import (
	"errors"
	"io"
	"net/http"

	"sqldash/config"
	"sqldash/pictures"
	repository "sqldash/repositories/account"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SavePicture(accountID uint, source io.Reader) *fiber.Error {
	picture, processError := pictures.Transform(source, int64(config.Server.MaximumImageUploadBytes))
	if processError != nil {
		status := http.StatusBadRequest
		if errors.Is(processError, pictures.UploadTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		return shortcuts.ServiceError(status, processError.Error())
	}

	reference, storageError := pictures.Store(picture.Base64, picture.MediaType)
	if storageError != nil {
		logger.Errorf(LogPrefix, SaveFailedLog, storageError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}

	if saveError := repository.SavePicture(accountID, reference, picture.MediaType); saveError != nil {
		logger.Errorf(LogPrefix, SaveFailedLog, saveError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}

	return nil
}
