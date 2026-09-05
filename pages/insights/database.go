package insights

import (
	"net/url"

	service "sqldash/services/insights"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func ForDatabase(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	data, dataError := service.GetOverview(name, context.Query(WindowParameter))
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context,
		meta.Crumb{Label: DatabasesLabel, URL: DatabasesPath},
		meta.Crumb{Label: name, URL: DatabasePath + url.PathEscape(name)},
		meta.Crumb{Label: InsightsLabel},
	)
	meta.SetSection(context, name, InsightsSection)

	return shortcuts.Render(context, OverviewTemplate, data)
}
