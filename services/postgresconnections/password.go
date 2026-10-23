package postgresconnections

import (
	"context"
	"errors"

	"sqldash/models"
	"sqldash/services/access"
	primarycredentials "sqldash/services/postgresconnections/primary"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func RotatePrimary(ctx context.Context, name string) *fiber.Error {
	fresh, record, err := authorize(ctx, name, models.ScopeReadWrite)
	if err != nil {
		return err
	}
	connection, findError := primarycredentials.Connection(fresh, record)
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		if err := primarycredentials.Ensure(fresh, record, access.AccountID(fresh)); err != nil {
			return shortcuts.ServiceError(fiber.StatusBadGateway, Refused)
		}
		return nil
	}
	if findError != nil {
		return shortcuts.ServiceError(fiber.StatusConflict, Refused)
	}
	_, rotateError := Rotate(fresh, name, connection.ID)
	return rotateError
}
