package middleware

import (
	"fmt"
	"net/url"
	"strings"

	"sqldash/permissions"
	"sqldash/services/access"
	explorer "sqldash/services/explorer"
	"sqldash/utils/logger"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

func palette(context fiber.Ctx) error {
	if context.Locals(meta.AccountKey) == nil {
		return context.Next()
	}

	heldAccess, _ := context.Locals(access.ContextKey).(*access.AccountAccess)
	groups := make([]meta.PaletteGroup, 0, 4)
	destinations := []meta.PaletteItem{{Label: "Settings", URL: "/settings"}}
	if heldAccess.Can(permissions.DatabaseView) {
		destinations = append(destinations, PaletteDestinations[0])
	}
	if heldAccess.Can(permissions.InsightsView) {
		destinations = append(destinations, PaletteDestinations[1])
	}
	groups = append(groups, meta.PaletteGroup{Label: PaletteGoLabel, Items: destinations})

	records, findError := access.Visible(heldAccess)
	if findError != nil {
		logger.Errorf(LogPrefix, PaletteListFailedLog, findError)
		context.Locals(meta.PaletteKey, groups)

		return context.Next()
	}

	databases := make([]meta.PaletteItem, 0, len(records))
	opened := databaseInPath(context.Path())
	held := false

	for _, record := range records {
		if !heldAccess.Can(permissions.DatabaseView) {
			continue
		}
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

	if held && heldAccess.Can(permissions.DataRead) {
		groups = append(groups, meta.PaletteGroup{
			Label: fmt.Sprintf(PaletteInsideLabel, opened),
			Items: sectionsOf(opened, heldAccess),
		})

		if tables := tablesOf(context, opened); len(tables) > 0 {
			groups = append(groups, meta.PaletteGroup{Label: PaletteTablesLabel, Items: tables})
		}
	}

	context.Locals(meta.PaletteKey, groups)

	return context.Next()
}

func sectionsOf(name string, held *access.AccountAccess) []meta.PaletteItem {
	base := DatabasePath + url.PathEscape(name)
	items := []meta.PaletteItem{}
	if held.Can(permissions.DatabaseView) {
		items = append(items, meta.PaletteItem{Label: PaletteOverviewLabel, URL: base})
	}
	if held.Can(permissions.DataRead) {
		items = append(items, meta.PaletteItem{Label: PaletteDataLabel, URL: base + ExplorePath})
	}
	if held.Can(permissions.SQLExecute) {
		items = append(items, meta.PaletteItem{Label: PaletteConsoleLabel, URL: base + ConsolePath})
	}
	if held.Can(permissions.InsightsView) {
		items = append(items, meta.PaletteItem{Label: PaletteInsightsLabel, URL: base + InsightsPath})
	}
	return items
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
