package settings

import (
	"net/http"
	"strings"

	"sqldash/database"
	repository "sqldash/repositories/account"
	"sqldash/utils/logger"
	"sqldash/utils/passwords"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveDetails(accountID uint, username string, email string) *fiber.Error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)

	if username == "" || email == "" {
		return shortcuts.ServiceError(http.StatusBadRequest, DetailsMissing)
	}

	held, findError := repository.FindByID(accountID)
	if findError != nil || held == nil {
		logger.Errorf(LogPrefix, LookupFailedLog, findError)
		return shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	held.Username = username
	held.Email = email

	if saveError := database.DB.Save(held).Error; saveError != nil {
		logger.Errorf(LogPrefix, SaveFailedLog, saveError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}

	return nil
}

func ChangePassword(accountID uint, current string, wanted string) *fiber.Error {
	held, findError := repository.FindByID(accountID)
	if findError != nil || held == nil {
		logger.Errorf(LogPrefix, LookupFailedLog, findError)
		return shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	if !passwords.Matches(current, held.PasswordHash) {
		return shortcuts.ServiceError(http.StatusBadRequest, CurrentWrong)
	}

	if acceptableError := passwords.Acceptable(wanted); acceptableError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, acceptableError.Error())
	}

	hashed, hashError := passwords.Hash(wanted)
	if hashError != nil {
		logger.Errorf(LogPrefix, SaveFailedLog, hashError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}

	held.PasswordHash = hashed

	if saveError := database.DB.Save(held).Error; saveError != nil {
		logger.Errorf(LogPrefix, SaveFailedLog, saveError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SaveRefused)
	}

	return nil
}
