package audit

import (
	"strconv"

	"sqldash/services/access"
	service "sqldash/services/audit"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Log(context fiber.Ctx) error {
	page, _ := strconv.Atoi(context.Query(PageParameter))
	filter := service.Filter{Engine: context.Query(EngineParameter)}
	if databaseID := context.Query(DatabaseIDParameter); databaseID != "" {
		parsedID, parseError := strconv.ParseUint(databaseID, 10, strconv.IntSize)
		if parseError != nil || parsedID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, service.InvalidFilter)
		}
		filter.DatabaseID = uint(parsedID)
	}

	data, dataError := service.PageFor(
		access.FromContext(context.Context()), context.Query(DatabaseParameter), context.Query(ActionParameter), page, filter,
	)

	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: data.Title})

	return shortcuts.Render(context, LogTemplate, data)
}
