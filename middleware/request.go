package middleware

import (
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

func request(context fiber.Ctx) error {
	context.Locals(meta.RequestKey, meta.BuildRequest(context))
	return context.Next()
}
