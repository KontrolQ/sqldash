package databases

import (
	"context"
	"net/http"

	"sqldash/database"
	repository "sqldash/repositories/database"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveSettings(requestContext context.Context, name string, blockReads bool, blockWrites bool, reason string, allowAttach bool, protected bool) *fiber.Error {
	held, findError := repository.FindByName(name)
	if findError != nil {
		logger.Errorf(LogPrefix, ListFailedLog, findError)
		return shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}

	if held == nil {
		return shortcuts.ServiceError(http.StatusNotFound, DatabaseMissing)
	}

	settings, settingsError := sqld.Configuration(requestContext, name)
	if settingsError != nil {
		logger.Errorf(LogPrefix, ConfigurationFailedLog, name, settingsError)
		return shortcuts.ServiceError(http.StatusBadGateway, SettingsRefused)
	}

	settings.BlockReads = blockReads
	settings.BlockWrites = blockWrites
	settings.AllowAttach = allowAttach

	if reason == "" {
		settings.BlockReason = nil
	} else {
		settings.BlockReason = &reason
	}

	if saveError := sqld.SetConfiguration(requestContext, name, *settings); saveError != nil {
		logger.Errorf(LogPrefix, ConfigurationFailedLog, name, saveError)
		return shortcuts.ServiceError(http.StatusBadGateway, SettingsRefused)
	}

	held.Protected = protected

	if saveError := database.DB.Save(held).Error; saveError != nil {
		logger.Errorf(LogPrefix, ConfigurationFailedLog, name, saveError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SettingsRefused)
	}

	return nil
}
