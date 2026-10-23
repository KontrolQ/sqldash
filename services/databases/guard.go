package databases

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"sqldash/models"
	repository "sqldash/repositories/database"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

var namePattern = regexp.MustCompile(NamePattern)

func NormalizeName(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

func AcceptableName(name string) (string, *fiber.Error) {
	return acceptableEngineName(name, models.EngineLibSQL)
}

func acceptableEngineName(name string, engine string) (string, *fiber.Error) {
	name = NormalizeName(name)
	if engine == models.EnginePostgres && (name == "postgres" || name == "template0" || name == "template1") {
		return "", shortcuts.ServiceError(http.StatusBadRequest, NameReserved)
	}

	if len(name) < MinimumNameLength {
		return "", shortcuts.ServiceError(http.StatusBadRequest, NameTooShort)
	}

	if len(name) > MaximumNameLength {
		return "", shortcuts.ServiceError(http.StatusBadRequest, NameTooLong)
	}

	if !namePattern.MatchString(name) {
		return "", shortcuts.ServiceError(http.StatusBadRequest, NameUnusable)
	}

	if slices.Contains(ReservedNames, name) {
		return "", shortcuts.ServiceError(http.StatusBadRequest, NameReserved)
	}

	existing, findError := repository.FindByName(name)
	if findError != nil {
		logger.Errorf(LogPrefix, ListFailedLog, findError)
		return "", shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}

	if existing != nil {
		return "", shortcuts.ServiceError(http.StatusConflict, NameTaken)
	}

	return name, nil
}
