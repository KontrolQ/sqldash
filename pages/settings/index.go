package settings

import (
	service "sqldash/services/settings"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Index(context fiber.Ctx) error {
	theme, _ := context.Locals(meta.ThemeKey).(string)

	data, dataError := service.GetIndexData(
		sessions.Remembered(context),
		theme,
	)

	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, IndexTemplate, data)
}
