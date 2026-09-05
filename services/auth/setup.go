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

func CreateFirstAccount(username string, email string, password string) (*models.Account, *fiber.Error) {
	needed, guardError := NeedsSetup()
	if guardError != nil {
		return nil, guardError
	}

	if !needed {
		return nil, shortcuts.ServiceError(http.StatusConflict, AlreadySetUp)
	}

	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)

	if username == "" || email == "" || password == "" {
		return nil, shortcuts.ServiceError(http.StatusBadRequest, DetailsMissing)
	}

	if acceptableError := passwords.Acceptable(password); acceptableError != nil {
		return nil, shortcuts.ServiceError(http.StatusBadRequest, acceptableError.Error())
	}

	hashed, hashError := passwords.Hash(password)
	if hashError != nil {
		logger.Errorf(LogPrefix, CreateFailedLog, hashError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	held := &models.Account{Username: username, Email: email, PasswordHash: hashed}

	if createError := repository.Create(held); createError != nil {
		logger.Errorf(LogPrefix, CreateFailedLog, createError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	return held, nil
}
