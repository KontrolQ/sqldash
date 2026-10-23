package postgresconnections

import (
	"sqldash/services/audit"
	service "sqldash/services/postgresconnections"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Index(context fiber.Ctx) error {
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "same-origin")
	context.Set("X-Content-Type-Options", "nosniff")
	name := context.Params("name")
	data, err := service.GetIndexData(context.Context(), name)
	if err != nil {
		return err
	}
	for action, permitted := range data.Actions {
		context.Locals(action, permitted)
	}
	context.Locals("DatabaseEngine", data.Engine)
	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: "Databases", URL: "/"}, meta.Crumb{Label: name, URL: "/databases/" + name}, meta.Crumb{Label: "Credentials"})
	meta.SetSection(context, name, "connections")
	if renderError := shortcuts.Render(context, "postgresconnections/index", data); renderError != nil {
		return renderError
	}
	if data.PasswordAvailable {
		audit.Note(context, name, audit.CredentialViewed, "")
	}
	return nil
}
