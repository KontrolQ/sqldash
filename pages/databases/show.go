package databases

import (
	"sqldash/services/audit"
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

	if data.PostgresCredentials != nil {
		context.Set("Cache-Control", "no-store")
		context.Set("Referrer-Policy", "same-origin")
		context.Set("X-Content-Type-Options", "nosniff")
	}

	for action, permitted := range data.Actions {
		context.Locals(action, permitted)
	}
	context.Locals("DatabaseEngine", data.Engine)
	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context,
		meta.Crumb{Label: DatabasesLabel, URL: DatabasesPath},
		meta.Crumb{Label: name},
	)
	meta.SetSection(context, name, OverviewSection)

	if renderError := shortcuts.Render(context, ShowTemplate, data); renderError != nil {
		return renderError
	}
	if data.PostgresCredentials != nil {
		audit.Note(context, name, audit.CredentialViewed, "")
	}
	return nil
}
