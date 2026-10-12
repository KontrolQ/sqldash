package administration

import (
	"fmt"

	"sqldash/services/members"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

type RoleMembersRequest struct {
	Accounts []uint `form:"accounts"`
	Action   string `form:"action"`
}

func ChangeRoleMembers(context fiber.Ctx) error {
	roleID := identifier(context)
	destination := fmt.Sprintf("/admin/roles?edit=%d&tab=users", roleID)
	asked, parseError := meta.Body[RoleMembersRequest](context)
	if parseError != nil || roleID == 0 || (asked.Action != "add" && asked.Action != "remove") {
		return complete(context, "role.users.changed", members.Invalid, destination)
	}
	return complete(context, "role.users.changed", members.ChangeRoleMembers(actor(context), roleID, asked.Accounts, asked.Action == "add"), destination)
}
