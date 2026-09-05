package auth

import (
	service "sqldash/services/auth"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Setup(context fiber.Ctx) error {
	needed, guardError := service.NeedsSetup()
	if guardError != nil {
		return guardError
	}

	if !needed {
		return shortcuts.Redirect(context, LoginRoute)
	}

	meta.SetPageTitle(context, service.SetupTitle)

	return shortcuts.Render(context, SetupTemplate, service.GateContext{
		Title: service.SetupTitle,
	})
}
