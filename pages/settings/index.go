package settings

import (
	service "sqldash/services/settings"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Index(context fiber.Ctx) error {
	data, dataError := service.GetIndexData(sessions.Remembered(context))
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: SettingsLabel})

	return shortcuts.Render(context, IndexTemplate, data)
}
