package auth

import (
	"github.com/gofiber/fiber/v3"
	"sqldash/services/members"
	"sqldash/utils/collections"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"
)

func Join(context fiber.Ctx) error    { return showLink(context, members.Invitation) }
func Recover(context fiber.Ctx) error { return showLink(context, members.Recovery) }
func showLink(context fiber.Ctx, kind string) error {
	context.Set("Referrer-Policy", "no-referrer")
	context.Set("Cache-Control", "no-store")
	link, linkError := members.FindLink(kind, context.Query("token"))
	if linkError != nil {
		return shortcuts.ServiceError(fiber.StatusBadRequest, members.LinkInvalid.Error())
	}
	title := "Accept invitation"
	if kind == members.Recovery {
		title = "Choose a new password"
	}
	meta.SetPageTitle(context, title)
	return shortcuts.Render(context, "auth/link", collections.Record[string, any]{"Title": title, "Email": link.Email, "Token": context.Query("token"), "Invitation": kind == members.Invitation})
}
