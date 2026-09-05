package auth

import (
	"net/http"
	"strings"

	"sqldash/models"
	repository "sqldash/repositories/account"
	"sqldash/utils/logger"
	"sqldash/utils/passwords"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SignIn(username string, password string) (*models.Account, *fiber.Error) {
	username = strings.TrimSpace(username)

	if username == "" || password == "" {
		return nil, shortcuts.ServiceError(http.StatusBadRequest, CredentialsWrong)
	}

	held, findError := repository.FindByUsername(username)
	if findError != nil {
		logger.Errorf(LogPrefix, LookupFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	if held == nil || !passwords.Matches(password, held.PasswordHash) {
		return nil, shortcuts.ServiceError(http.StatusUnauthorized, CredentialsWrong)
	}

	return held, nil
}
