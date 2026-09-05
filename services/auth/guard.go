package auth

import (
	"net/http"

	repository "sqldash/repositories/account"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func NeedsSetup() (bool, *fiber.Error) {
	total, countError := repository.Count()
	if countError != nil {
		logger.Errorf(LogPrefix, CountFailedLog, countError)
		return false, shortcuts.ServiceError(http.StatusInternalServerError, AccountUnavailable)
	}

	return total == 0, nil
}
