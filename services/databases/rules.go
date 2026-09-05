package databases

import (
	"net"
	"net/http"
	"strings"

	"sqldash/models"
	repository "sqldash/repositories/accessrule"
	databaseRepository "sqldash/repositories/database"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func AllowNetwork(databaseName string, network string, note string) *fiber.Error {
	network = strings.TrimSpace(network)

	if _, _, parseError := net.ParseCIDR(network); parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, NetworkUnusable)
	}

	held, findError := databaseRepository.FindByName(databaseName)
	if findError != nil || held == nil {
		return shortcuts.ServiceError(http.StatusNotFound, DatabaseMissing)
	}

	record := &models.AccessRule{
		DatabaseName: databaseName,
		Network:      network,
		Note:         strings.TrimSpace(note),
	}

	if createError := repository.Create(record); createError != nil {
		logger.Errorf(LogPrefix, RuleFailedLog, databaseName, createError)
		return shortcuts.ServiceError(http.StatusInternalServerError, NetworkRefused)
	}

	return nil
}

func RemoveNetwork(databaseName string, identifier uint) *fiber.Error {
	rules, findError := repository.ForDatabase(databaseName)
	if findError != nil {
		logger.Errorf(LogPrefix, RuleFailedLog, databaseName, findError)
		return shortcuts.ServiceError(http.StatusInternalServerError, NetworkRefused)
	}

	for _, rule := range rules {
		if rule.ID != identifier {
			continue
		}

		if deleteError := repository.Delete(&rule); deleteError != nil {
			logger.Errorf(LogPrefix, RuleFailedLog, databaseName, deleteError)
			return shortcuts.ServiceError(http.StatusInternalServerError, NetworkRefused)
		}

		return nil
	}

	return shortcuts.ServiceError(http.StatusNotFound, NetworkMissing)
}
