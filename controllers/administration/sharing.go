package administration

import (
	"fmt"
	"net/url"

	"sqldash/services/members"

	"github.com/gofiber/fiber/v3"
)

func ShareDatabase(context fiber.Ctx) error {
	var recipient uint
	fmt.Sscan(context.FormValue("account_id"), &recipient)
	add := context.FormValue("action") != "remove"
	action := "database.unshared"
	if add {
		action = "database.shared"
	}
	return complete(context, action, members.ShareDatabase(actor(context), context.Params("name"), recipient, add), "/databases/"+url.PathEscape(context.Params("name"))+"/sharing")
}
