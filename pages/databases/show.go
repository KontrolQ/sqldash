package databases

import (
	service "sqldash/services/databases"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Show(context fiber.Ctx) error {
	data, dataError := service.GetShowData(
		context.Context(),
		context.Params(NameParameter),
		"",
	)

	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, ShowTemplate, data)
}
