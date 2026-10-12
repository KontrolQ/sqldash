package settings

import (
	"fmt"
	"net/http"

	"sqldash/config"
	repository "sqldash/repositories/account"
	"sqldash/services/auth"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetIndexData(accountID uint) (*IndexContext, *fiber.Error) {
	held, findError := repository.FindByID(accountID)
	if findError != nil {
		logger.Errorf(LogPrefix, LookupFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	if held == nil {
		return nil, shortcuts.ServiceError(http.StatusUnauthorized, AccountUnavailable)
	}

	history, historyError := repository.PictureHistory(accountID)
	if historyError != nil {
		logger.Errorf(LogPrefix, LookupFailedLog, historyError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	account := auth.ToAccountView(held)

	return &IndexContext{
		Title:                   IndexTitle,
		Username:                account.Username,
		DisplayName:             account.DisplayName,
		Email:                   account.Email,
		ProfilePicture:          account.ProfilePicture,
		PictureUploadLimit:      fmt.Sprintf(PictureUploadLimit, float64(config.Server.MaximumImageUploadBytes)/(1024*1024)),
		PictureHistory:          pictureHistory(held, history),
		MaximumImageUploadBytes: config.Server.MaximumImageUploadBytes,
	}, nil
}
