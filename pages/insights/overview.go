package insights

import (
	"sqldash/services/access"
	service "sqldash/services/insights"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Overview(context fiber.Ctx) error {
	scope, scopeError := access.MonitoringScope(context.Context())
	if scopeError != nil {
		return scopeError
	}
	data, dataError := service.GetOverview("", context.Query(WindowParameter), scope...)
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: InsightsLabel})

	return shortcuts.Render(context, OverviewTemplate, data)
}
