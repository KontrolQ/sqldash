package databases

import (
	"context"
	"errors"
	"net/http"

	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func reserveCreation(ctx context.Context, actor *access.AccountAccess, asked string, engine string) (*models.Database, *models.DatabaseOperation, *fiber.Error) {
	for attempt := 0; attempt < MaximumCreationNameAttempts; attempt++ {
		candidate, candidateError := creationNameCandidate(asked, actor.Account.Username, attempt)
		if candidateError != nil {
			return nil, nil, shortcuts.ServiceError(http.StatusInternalServerError, CreateRefused)
		}
		name, nameError := acceptableEngineName(candidate, engine)
		if nameError == nil {
			record, operation, reserveError := reserveDatabase(ctx, name, engine, actor.Account.ID, "create", 0, operationPayload{})
			if reserveError == nil {
				return record, operation, nil
			}
			nameError = reserveError
		}
		if nameError.Code != http.StatusConflict || nameError.Message != NameTaken {
			return nil, nil, nameError
		}
		if attempt == 0 {
			if visibilityError := creationNameConflict(actor, candidate); visibilityError != nil {
				return nil, nil, visibilityError
			}
		}
	}
	return nil, nil, shortcuts.ServiceError(http.StatusServiceUnavailable, NameAllocationRefused)
}

func creationNameConflict(actor *access.AccountAccess, name string) *fiber.Error {
	record, findingError := access.Database(actor, name)
	if errors.Is(findingError, gorm.ErrRecordNotFound) {
		return nil
	}
	if findingError != nil {
		return shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}
	if actor.CanDatabase(record, permissions.DatabaseView) {
		return shortcuts.ServiceError(http.StatusConflict, NameTaken)
	}
	return nil
}
