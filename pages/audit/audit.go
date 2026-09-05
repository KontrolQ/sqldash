package audit

import (
	"strconv"

	service "sqldash/services/audit"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Log(context fiber.Ctx) error {
	page, _ := strconv.Atoi(context.Query(PageParameter))

	data, dataError := service.Page(
		context.Query(DatabaseParameter), context.Query(ActionParameter), page,
	)

	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: data.Title})

	return shortcuts.Render(context, LogTemplate, data)
}
