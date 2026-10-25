package middleware

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"net/url"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	explorer "sqldash/services/explorer"
	"sqldash/utils/logger"
	"sqldash/utils/meta"
	"strings"
)

func palette(context fiber.Ctx) error {
	if context.Locals(meta.AccountKey) == nil {
		return context.Next()
	}
	heldAccess, _ := context.Locals(access.ContextKey).(*access.AccountAccess)
	groups := make([]meta.PaletteGroup, 0, 4)
	destinations := []meta.PaletteItem{{Label: "Settings", URL: "/settings"}}
	if heldAccess.CanAny(permissions.DatabaseView) {
		destinations = append(destinations, PaletteDestinations[0])
	}
	if heldAccess.CanAny(permissions.InsightsView) {
		destinations = append(destinations, PaletteDestinations[1])
	}
	groups = append(groups, meta.PaletteGroup{Label: PaletteGoLabel, Items: destinations})
	records, findError := access.VisibleFor(heldAccess, permissions.DatabaseView)
	if findError != nil {
		logger.Errorf(LogPrefix, PaletteListFailedLog, findError)
		context.Locals(meta.PaletteKey, groups)
		return context.Next()
	}
	destinations = make([]meta.PaletteItem, 0, len(records))
	opened := databaseInPath(context.Path())
	var current *models.Database
	for index := range records {
		record := &records[index]
		destinations = append(destinations, meta.PaletteItem{Label: record.Name, Note: record.Kind(), URL: DatabasePath + url.PathEscape(record.Name)})
		if record.Name == opened {
			current = record
		}
	}
	groups = append(groups, meta.PaletteGroup{Label: PaletteDatabasesLabel, Items: destinations})
	if current != nil && heldAccess.CanDatabase(current, permissions.DataRead) && current.Available() {
		groups = append(groups, meta.PaletteGroup{Label: fmt.Sprintf(PaletteInsideLabel, opened), Items: sectionsOf(current, heldAccess)})
		if current.Kind() != models.EngineRedis {
			if tables := tablesOf(context, opened); len(tables) > 0 {
				groups = append(groups, meta.PaletteGroup{Label: PaletteTablesLabel, Items: tables})
			}
		}
	}
	context.Locals(meta.PaletteKey, groups)
	return context.Next()
}

func sectionsOf(record *models.Database, held *access.AccountAccess) []meta.PaletteItem {
	base := DatabasePath + url.PathEscape(record.Name)
	items := []meta.PaletteItem{}
	for _, destination := range []struct{ permission, label, suffix string }{
		{permissions.DatabaseView, PaletteOverviewLabel, ""},
		{permissions.DataRead, PaletteDataLabel, ExplorePath},
		{permissions.SQLExecute, PaletteConsoleLabel, ConsolePath},
		{permissions.InsightsView, PaletteInsightsLabel, InsightsPath},
	} {
		if held.CanDatabase(record, destination.permission) {
			items = append(items, meta.PaletteItem{Label: destination.label, URL: base + destination.suffix})
		}
	}
	if record.Kind() == models.EngineRedis {
		for _, destination := range []struct{ permission, label, suffix string }{{permissions.RedisSlowLogView, "Slow log", "/slow-log"}, {permissions.RedisMonitorView, "Command monitor", "/monitor"}, {permissions.RedisSubscribe, "Pub/Sub", "/pubsub"}, {permissions.RedisActivityView, "Connections", "/connections"}} {
			if held.CanDatabase(record, destination.permission) {
				items = append(items, meta.PaletteItem{Label: destination.label, URL: base + destination.suffix})
			}
		}
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
		address := base + url.QueryEscape(record.Name)
		if record.Schema != "" {
			address += "&schema=" + url.QueryEscape(record.Schema)
		}
		items = append(items, meta.PaletteItem{Label: record.Name, Note: note, URL: address})
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
