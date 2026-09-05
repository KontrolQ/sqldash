package audit

import (
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
	Record(ActorOf(context), databaseName, action, detail)
}
