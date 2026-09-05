package databases

import (
	"context"
	"net/http"

	"sqldash/hosting"
	repository "sqldash/repositories/database"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Delete(requestContext context.Context, name string) *fiber.Error {
	held, findError := repository.FindByName(name)
	if findError != nil {
		logger.Errorf(LogPrefix, ListFailedLog, findError)
		return shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}

	if held == nil {
		return shortcuts.ServiceError(http.StatusNotFound, DatabaseMissing)
	}

	if held.Protected {
		return shortcuts.ServiceError(http.StatusConflict, DatabaseHeld)
	}

	if deleteError := sqld.Delete(requestContext, name); deleteError != nil {
		logger.Errorf(LogPrefix, DeleteFailedLog, name, deleteError)
		return shortcuts.ServiceError(http.StatusBadGateway, DeleteRefused)
	}

	if forgetError := repository.Delete(held); forgetError != nil {
		logger.Errorf(LogPrefix, DeleteFailedLog, name, forgetError)
		return shortcuts.ServiceError(http.StatusInternalServerError, DeleteRefused)
	}

	go hosting.Withdraw(context.WithoutCancel(requestContext), name)

	return nil
}
