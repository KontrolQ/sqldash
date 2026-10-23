package postgresconnections

import (
	"context"

	"sqldash/models"
	primarycredentials "sqldash/services/postgresconnections/primary"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func mutateCredential(ctx context.Context, record *models.Database, accountID uint, mutate func(context.Context) *fiber.Error) *fiber.Error {
	var actionError *fiber.Error
	err := primarycredentials.Mutate(ctx, record, accountID, func(fenced context.Context) error {
		actionError = mutate(fenced)
		if actionError != nil {
			return actionError
		}
		return nil
	})
	if actionError != nil {
		return actionError
	}
	if err != nil {
		return shortcuts.ServiceError(fiber.StatusConflict, Refused)
	}
	return nil
}
