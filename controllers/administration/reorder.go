package administration

import (
	"sqldash/services/audit"
	"sqldash/services/members"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

type ReorderRequest struct {
	Roles []uint `json:"roles"`
}

func ReorderRoles(context fiber.Ctx) error {
	asked, parseError := meta.Body[ReorderRequest](context)
	if parseError != nil {
		return fiber.ErrBadRequest
	}
	if reorderError := members.ReorderRoles(actor(context), asked.Roles); reorderError != nil {
		return fiber.ErrForbidden
	}
	audit.Note(context, "", "role.reordered", "")
	return context.SendStatus(fiber.StatusNoContent)
}
