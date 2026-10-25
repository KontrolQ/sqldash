package router

import (
	"sqldash/models"

	"github.com/gofiber/fiber/v3"
)

func redisPage(relational, keyValue fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		record, _ := ctx.Locals("DatabaseRecord").(*models.Database)
		if record != nil && record.Kind() == models.EngineRedis {
			return keyValue(ctx)
		}
		return relational(ctx)
	}
}
