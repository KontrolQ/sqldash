package middleware

import (
	"sqldash/sessions"

	"github.com/gofiber/fiber/v3"
)

func Initialize(application *fiber.App) {
	application.Use(sessions.Handler)
	application.Use(request)
	application.Use(chrome)
	application.Use(account)
}
