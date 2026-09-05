package middleware

import (
	"slices"

	"sqldash/config"
	"sqldash/sessions"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

func chrome(context fiber.Ctx) error {
	context.Locals(meta.VersionKey, config.AppVersion)
	context.Locals(meta.ThemeKey, chosenTheme(context))

	problem, done := sessions.TakeMessages(context)
	context.Locals(meta.ProblemKey, problem)
	context.Locals(meta.DoneKey, done)

	return context.Next()
}

func chosenTheme(context fiber.Ctx) string {
	held := context.Cookies(ThemeCookie)

	if slices.Contains(Themes, held) {
		return held
	}

	return ThemeSystem
}
