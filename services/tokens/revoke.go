package tokens

import (
	"net/http"
	"time"

	repository "sqldash/repositories/token"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Revoke(databaseName string, identifier string) *fiber.Error {
	held, findError := repository.FindByIdentifier(identifier)
	if findError != nil {
		logger.Errorf(LogPrefix, RevokeFailedLog, identifier, findError)
		return shortcuts.ServiceError(http.StatusInternalServerError, RevokeRefused)
	}

	if held == nil || held.DatabaseName != databaseName {
		return shortcuts.ServiceError(http.StatusNotFound, TokenMissing)
	}

	if held.RevokedAt != nil {
		return nil
	}

	now := time.Now()
	held.RevokedAt = &now

	if saveError := repository.Save(held); saveError != nil {
		logger.Errorf(LogPrefix, RevokeFailedLog, identifier, saveError)
		return shortcuts.ServiceError(http.StatusInternalServerError, RevokeRefused)
	}

	return nil
}
