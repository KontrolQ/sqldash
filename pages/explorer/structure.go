package explorer

import (
	service "sqldash/services/explorer"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Structure(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	data, dataError := service.Structure(context.Context(), name, context.Query(TableParameter))
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	describe(context, name, StructureLabel, StructureSection)

	return shortcuts.Render(context, StructureTemplate, data)
}
