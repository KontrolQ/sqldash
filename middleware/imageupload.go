package middleware

import (
	"sqldash/config"
	"sqldash/sessions"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func ImageUpload(field string, fallback string, next fiber.Handler) fiber.Handler {
	return func(context fiber.Ctx) error {
		file, fileError := context.FormFile(field)
		if fileError != nil {
			sessions.Complain(context, ImageMissing)
			return shortcuts.RedirectToPath(context, fallback)
		}
		if config.Server.MaximumImageUploadBytes > 0 && file.Size > int64(config.Server.MaximumImageUploadBytes) {
			sessions.Complain(context, ImageUploadTooLarge)
			return shortcuts.RedirectToPath(context, fallback)
		}
		return next(context)
	}
}
