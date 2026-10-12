package administration

import (
	"github.com/gofiber/fiber/v3"
	"sqldash/services/members"
	"sqldash/utils/meta"
)

type RoleRequest struct {
	Name        string   `form:"name"`
	Color       string   `form:"color"`
	Permissions []string `form:"permissions"`
}

func SaveRole(context fiber.Ctx) error {
	if context.Params("identifier") != "" && identifier(context) == 0 {
		return complete(context, RoleSaved, members.Invalid, RolesPath)
	}
	asked, parseError := meta.Body[RoleRequest](context)
	if parseError != nil {
		return complete(context, RoleSaved, members.Invalid, RolesPath)
	}
	return complete(context, RoleSaved, members.SaveRole(actor(context), identifier(context), asked.Name, asked.Color, 0, asked.Permissions), RolesPath)
}
func DeleteRole(context fiber.Ctx) error {
	return complete(context, RoleDeleted, members.DeleteRole(actor(context), identifier(context)), RolesPath)
}
