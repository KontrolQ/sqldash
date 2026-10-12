package administration

import (
	"sqldash/services/members"

	"github.com/gofiber/fiber/v3"
)

func TransferOwnership(context fiber.Ctx) error {
	return complete(context, "ownership.requested", members.TransferOwnership(actor(context), identifier(context), context.FormValue("password")), UsersPath)
}

func RespondOwnership(context fiber.Ctx) error {
	action := context.FormValue("action")
	event := map[string]string{"accept": "ownership.accepted", "decline": "ownership.declined", "cancel": "ownership.cancelled"}[action]
	return complete(context, event, members.RespondOwnership(actor(context), identifier(context), action, context.FormValue("password")), "/settings")
}
