package databases

import (
	"sqldash/models"
	service "sqldash/services/databases"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Index(context fiber.Ctx) error {
	return renderIndex(context, context.Query("engine"), nil)
}

func RemoteCloneFailure(context fiber.Ctx, form *service.RemoteCloneForm) error {
	return renderIndex(context, models.EnginePostgres, form)
}

func renderIndex(context fiber.Ctx, engine string, form *service.RemoteCloneForm) error {
	data, dataError := service.GetIndexDataForEngine(context.Context(), engine)
	if dataError != nil {
		return dataError
	}

	data.RemoteCloneForm = form
	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: DatabasesLabel})

	return shortcuts.Render(context, IndexTemplate, data)
}
