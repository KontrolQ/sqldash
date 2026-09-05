package auth

import (
	service "sqldash/services/auth"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Login(context fiber.Ctx) error {
	needed, guardError := service.NeedsSetup()
	if guardError != nil {
		return guardError
	}

	if needed {
		return shortcuts.Redirect(context, SetupRoute)
	}

	if sessions.Remembered(context) != 0 {
		return shortcuts.Redirect(context, HomeRoute)
	}

	meta.SetPageTitle(context, service.LoginTitle)

	return shortcuts.Render(context, LoginTemplate, service.GateContext{
		Title:   service.LoginTitle,
		Problem: context.Query(ProblemParameter),
	})
}
