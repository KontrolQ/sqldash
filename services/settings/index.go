package settings

import (
	"net/http"

	repository "sqldash/repositories/account"
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

	return &IndexContext{
		Title:    IndexTitle,
		Username: held.Username,
		Email:    held.Email,
	}, nil
}
