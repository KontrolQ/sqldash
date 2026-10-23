package insights

import (
	service "sqldash/services/insights"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Overview(context fiber.Ctx) error {
	data, dataError := service.GetScopedOverview(context.Context(), "", context.Query(WindowParameter), context.Query("engine"))
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: InsightsLabel})

	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "no-referrer")
	return shortcuts.Render(context, OverviewTemplate, data)
}
