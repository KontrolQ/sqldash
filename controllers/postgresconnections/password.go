package postgresconnections

import (
	page "sqldash/pages/postgresconnections"
	"sqldash/permissions"
	"sqldash/services/audit"
	databaseservice "sqldash/services/databases"
	service "sqldash/services/postgresconnections"
	"sqldash/sessions"

	"github.com/gofiber/fiber/v3"
)

func RotatePrimary(context fiber.Ctx) error {
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "no-referrer")
	name := context.Params("name")
	_, record, err := databaseservice.Authorize(context.Context(), name, permissions.PostgresCredentialsManage, true)
	if err != nil {
		sessions.Complain(context, err.Message)
		return back(context)
	}
	if err := service.RotatePrimary(context.Context(), name); err != nil {
		sessions.Complain(context, err.Message)
	} else {
		audit.RecordDatabase(audit.ActorOf(context), record, service.RotatedAction, "")
		sessions.Report(context, service.Rotated)
	}
	return back(context)
}

func Reveal(context fiber.Ctx) error {
	if revealError := page.Index(context); revealError != nil {
		return revealError
	}
	audit.Note(context, context.Params("name"), audit.CredentialRevealed, "")
	return nil
}
