package databases

import (
	service "sqldash/services/databases"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Show(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	data, dataError := service.GetShowData(context.Context(), name, "")
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context,
		meta.Crumb{Label: DatabasesLabel, URL: DatabasesPath},
		meta.Crumb{Label: name},
	)
	meta.SetSection(context, name, OverviewSection)

	return shortcuts.Render(context, ShowTemplate, data)
}
