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
	_, createError := CreateEngineRecord(ctx, asked, engine)
	return createError
}

func CreateEngineRecord(ctx context.Context, asked string, engine string) (*models.Database, *fiber.Error) {
	engine = selectedEngine(engine)
	fresh, actor, authorityError := creationActor(ctx, engine, permissions.DatabaseCreate)
	if authorityError != nil {
		return nil, authorityError
	}
	record, operation, reserveError := reserveCreation(fresh, actor, NormalizeName(asked), engine)
	if reserveError != nil {
		return nil, reserveError
	}
	if createError := executeOperation(fresh, record, operation); createError != nil {
		return nil, createError
	}
	return record, nil
}
