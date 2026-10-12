package insights

import (
	"net/url"
	"strconv"

	"sqldash/services/access"
	service "sqldash/services/insights"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Queries(context fiber.Ctx) error {
	scope, scopeError := access.MonitoringScope(context.Context())
	if scopeError != nil {
		return scopeError
	}
	data, dataError := service.GetQueries(asked(context, ""), scope...)
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context,
		meta.Crumb{Label: InsightsLabel, URL: InsightsPath},
		meta.Crumb{Label: QueriesLabel},
	)

	return shortcuts.Render(context, QueriesTemplate, data)
}

func QueriesForDatabase(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	scope, scopeError := access.MonitoringScope(context.Context())
	if scopeError != nil {
		return scopeError
	}
	data, dataError := service.GetQueries(asked(context, name), scope...)
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context,
		meta.Crumb{Label: DatabasesLabel, URL: DatabasesPath},
		meta.Crumb{Label: name, URL: DatabasePath + url.PathEscape(name)},
		meta.Crumb{Label: QueriesLabel},
	)
	meta.SetSection(context, name, QueriesSection)

	return shortcuts.Render(context, QueriesTemplate, data)
}

func asked(context fiber.Ctx, name string) service.QueriesAsk {
	page, _ := strconv.Atoi(context.Query(PageParameter))

	return service.QueriesAsk{
		Database:  name,
		Window:    context.Query(WindowParameter),
		Sort:      context.Query(SortParameter),
		Direction: context.Query(DirectionParameter),
		Page:      page,
	}
}
