package explorer

import (
	"net/http"
	"strings"

	"sqldash/models"
	repository "sqldash/repositories/snippet"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Snippets(databaseName string) []SnippetView {
	records, findError := repository.ForDatabase(databaseName)
	if findError != nil {
		logger.Errorf(LogPrefix, SnippetsFailedLog, databaseName, findError)
		return nil
	}

	views := make([]SnippetView, 0, len(records))

	for _, record := range records {
		views = append(views, SnippetView{
			ID:        record.ID,
			Name:      record.Name,
			Statement: record.Statement,
		})
	}

	return views
}

func SnippetStatement(databaseName string, identifier uint) string {
	held, findError := repository.FindByID(identifier)
	if findError != nil || held == nil || held.Database != databaseName {
		return ""
	}

	return held.Statement
}

func SaveSnippet(databaseName string, identifier uint, name string, statement string) *fiber.Error {
	name = strings.TrimSpace(name)
	statement = strings.TrimSpace(statement)

	if statement == "" {
		return shortcuts.ServiceError(http.StatusBadRequest, SnippetEmpty)
	}

	if name == "" {
		name = firstLine(statement)
	}

	if identifier != 0 {
		held, findError := repository.FindByID(identifier)
		if findError == nil && held != nil && held.Database == databaseName {
			held.Name = name
			held.Statement = statement

			if saveError := repository.Save(held); saveError != nil {
				logger.Errorf(LogPrefix, SnippetsFailedLog, databaseName, saveError)
				return shortcuts.ServiceError(http.StatusInternalServerError, SnippetRefused)
			}

			return nil
		}
	}

	record := &models.Snippet{Database: databaseName, Name: name, Statement: statement}

	if createError := repository.Create(record); createError != nil {
		logger.Errorf(LogPrefix, SnippetsFailedLog, databaseName, createError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SnippetRefused)
	}

	return nil
}

func RemoveSnippet(databaseName string, identifier uint) *fiber.Error {
	held, findError := repository.FindByID(identifier)
	if findError != nil || held == nil || held.Database != databaseName {
		return shortcuts.ServiceError(http.StatusNotFound, SnippetMissing)
	}

	if deleteError := repository.Delete(identifier); deleteError != nil {
		logger.Errorf(LogPrefix, SnippetsFailedLog, databaseName, deleteError)
		return shortcuts.ServiceError(http.StatusInternalServerError, SnippetRefused)
	}

	return nil
}

func firstLine(statement string) string {
	for _, line := range strings.Split(statement, "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, CommentMarker) {
			continue
		}

		if len(trimmed) > SnippetNameLength {
			return trimmed[:SnippetNameLength] + Ellipsis
		}

		return trimmed
	}

	return UntitledSnippet
}
