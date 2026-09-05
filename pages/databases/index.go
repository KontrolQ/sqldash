package databases

import (
	service "sqldash/services/databases"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Index(context fiber.Ctx) error {
	data, dataError := service.GetIndexData(context.Query(ProblemParameter))
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, IndexTemplate, data)
}
