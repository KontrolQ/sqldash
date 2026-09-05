package insights

import (
	service "sqldash/services/insights"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Overview(context fiber.Ctx) error {
	data, dataError := service.GetOverview("", context.Query(WindowParameter))
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, OverviewTemplate, data)
}
