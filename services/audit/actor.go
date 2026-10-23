package audit

import (
	"sqldash/models"
	"sqldash/services/auth"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

func ActorOf(context fiber.Ctx) string {
	held, matched := context.Locals(meta.AccountKey).(auth.AccountView)
	if !matched || held.Username == "" {
		return UnknownActor
	}

	return held.Username
}

func Note(context fiber.Ctx, databaseName string, action string, detail string) {
	if record, matched := context.Locals("DatabaseRecord").(*models.Database); matched && record != nil && record.Name == databaseName {
		NoteDatabase(context, record, action, detail)
		return
	}
	Record(ActorOf(context), databaseName, action, detail)
}

func NoteDatabase(context fiber.Ctx, record *models.Database, action string, detail string) {
	RecordDatabase(ActorOf(context), record, action, detail)
}
