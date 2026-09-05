package meta

import "github.com/gofiber/fiber/v3"

func SetPageTitle(context fiber.Ctx, title string) {
	context.Locals(TitleKey, title)
}
