package postgresmanagement

import "github.com/gofiber/fiber/v3"

func Connections(context fiber.Ctx) error {
	context.Set("X-Content-Type-Options", "nosniff")
	return Activity(context)
}
