package middleware

import "github.com/gofiber/fiber/v3"

func Initialize(application *fiber.App) {
	application.Use(request)
	application.Use(chrome)
}
