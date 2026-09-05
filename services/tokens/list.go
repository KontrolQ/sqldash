package tokens

import (
	"net/http"
	"time"

	"sqldash/models"
	repository "sqldash/repositories/token"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func ForDatabase(name string) ([]TokenView, *fiber.Error) {
	records, findError := repository.ForDatabase(name)
	if findError != nil {
		logger.Errorf(LogPrefix, ListFailedLog, name, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}

	now := time.Now()
	views := make([]TokenView, 0, len(records))

	for _, record := range records {
		views = append(views, TokenView{
			Identifier: record.Identifier,
			Label:      record.Label,
			Prefix:     record.Prefix,
			Scope:      string(record.Scope),
			Revoked:    record.RevokedAt != nil,
			Expired:    record.ExpiresAt != nil && now.After(*record.ExpiresAt),
			CreatedAt:  record.CreatedAt,
			LastUsedAt: record.LastUsedAt,
		})
	}

	return views, nil
}

func scopeFrom(asked string) (models.TokenScope, *fiber.Error) {
	switch models.TokenScope(asked) {
	case models.ScopeReadOnly:
		return models.ScopeReadOnly, nil
	case models.ScopeReadWrite:
		return models.ScopeReadWrite, nil
	default:
		return "", shortcuts.ServiceError(http.StatusBadRequest, ScopeUnknown)
	}
}
