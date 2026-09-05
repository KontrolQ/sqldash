package middleware

import (
	"net/url"
	"strings"

	repository "sqldash/repositories/database"
	explorer "sqldash/services/explorer"
	"sqldash/utils/logger"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

func palette(context fiber.Ctx) error {
	if context.Locals(meta.AccountKey) == nil {
		return context.Next()
	}

	groups := make([]meta.PaletteGroup, 0, 4)
	groups = append(groups, meta.PaletteGroup{Label: PaletteGoLabel, Items: PaletteDestinations})

	records, findError := repository.All()
	if findError != nil {
		logger.Errorf(LogPrefix, PaletteListFailedLog, findError)
		context.Locals(meta.PaletteKey, groups)

		return context.Next()
	}

	databases := make([]meta.PaletteItem, 0, len(records))
	opened := databaseInPath(context.Path())
	held := false

	for _, record := range records {
		databases = append(databases, meta.PaletteItem{
			Label: record.Name,
			Note:  PaletteDatabaseNote,
			URL:   DatabasePath + url.PathEscape(record.Name),
		})

		if record.Name == opened {
			held = true
		}
	}

	groups = append(groups, meta.PaletteGroup{Label: PaletteDatabasesLabel, Items: databases})

	if held {
		groups = append(groups, meta.PaletteGroup{Label: opened, Items: sectionsOf(opened)})

		if tables := tablesOf(context, opened); len(tables) > 0 {
			groups = append(groups, meta.PaletteGroup{Label: PaletteTablesLabel, Items: tables})
		}
	}

	context.Locals(meta.PaletteKey, groups)

	return context.Next()
}

func sectionsOf(name string) []meta.PaletteItem {
	base := DatabasePath + url.PathEscape(name)

	return []meta.PaletteItem{
		{Label: PaletteOverviewLabel, Note: name, URL: base},
		{Label: PaletteDataLabel, Note: name, URL: base + ExplorePath},
		{Label: PaletteConsoleLabel, Note: name, URL: base + ConsolePath},
		{Label: PaletteInsightsLabel, Note: name, URL: base + InsightsPath},
	}
}

func tablesOf(context fiber.Ctx, name string) []meta.PaletteItem {
	records, tableError := explorer.Tables(context.Context(), name)
	if tableError != nil {
		return nil
	}

	base := DatabasePath + url.PathEscape(name) + ExplorePath + TableQuery

	items := make([]meta.PaletteItem, 0, len(records))

	for _, record := range records {
		note := PaletteTableNote
		if record.IsView {
			note = PaletteViewNote
		}

		items = append(items, meta.PaletteItem{
			Label: record.Name,
			Note:  note,
			URL:   base + url.QueryEscape(record.Name),
		})
	}

	return items
}

func databaseInPath(path string) string {
	if !strings.HasPrefix(path, DatabasePath) {
		return ""
	}

	rest := strings.TrimPrefix(path, DatabasePath)
	if cut := strings.IndexByte(rest, '/'); cut >= 0 {
		rest = rest[:cut]
	}

	name, unescapeError := url.PathUnescape(rest)
	if unescapeError != nil {
		return ""
	}

	return name
}
