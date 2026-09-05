package middleware

import (
	"sqldash/config"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

func chrome(context fiber.Ctx) error {
	context.Locals(meta.VersionKey, config.AppVersion)
	return context.Next()
}
