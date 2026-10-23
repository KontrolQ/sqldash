package databases

import (
	"context"

	"sqldash/models"
	"sqldash/permissions"

	"github.com/gofiber/fiber/v3"
)

func Create(ctx context.Context, asked string) *fiber.Error {
	return CreateEngine(ctx, asked, models.EngineLibSQL)
}

func CreateEngine(ctx context.Context, asked string, engine string) *fiber.Error {
	engine = selectedEngine(engine)
	fresh, actor, authorityError := creationActor(ctx, engine, permissions.DatabaseCreate)
	if authorityError != nil {
		return authorityError
	}
	name, nameError := acceptableEngineName(asked, engine)
	if nameError != nil {
		return nameError
	}
	record, operation, reserveError := reserveDatabase(fresh, name, engine, actor.Account.ID, "create", 0, operationPayload{})
	if reserveError != nil {
		return reserveError
	}
	return executeOperation(fresh, record, operation)
}
