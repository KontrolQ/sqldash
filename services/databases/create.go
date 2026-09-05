package databases

import (
	"context"
	"net/http"

	"sqldash/models"
	repository "sqldash/repositories/database"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Create(requestContext context.Context, asked string) *fiber.Error {
	name, guardError := AcceptableName(asked)
	if guardError != nil {
		return guardError
	}

	if createError := sqld.Create(requestContext, name); createError != nil {
		logger.Errorf(LogPrefix, CreateFailedLog, name, createError)
		return shortcuts.ServiceError(http.StatusBadGateway, CreateRefused)
	}

	if registerError := repository.Create(&models.Database{Name: name}); registerError != nil {
		logger.Errorf(LogPrefix, RegisterLog, name, registerError)
		return shortcuts.ServiceError(http.StatusInternalServerError, CreateRefused)
	}

	return nil
}
