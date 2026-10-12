package administration

import (
	"errors"
	"fmt"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/services/auth"
	"sqldash/services/mailing"
	"sqldash/services/members"
	"sqldash/utils/collections"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func held(context fiber.Ctx) *access.AccountAccess {
	actor, _ := context.Locals(access.ContextKey).(*access.AccountAccess)
	return actor
}

func pageFailure(failure error) error {
	if errors.Is(failure, members.Refused) {
		return fiber.ErrForbidden
	}
	if errors.Is(failure, members.Invalid) {
		return fiber.ErrNotFound
	}
	return failure
}

func Users(context fiber.Ctx) error {
	var editing uint
	fmt.Sscan(context.Query("edit"), &editing)
	var roleID uint
	fmt.Sscan(context.Query("role"), &roleID)
	issued := members.TakeIssuedLink(held(context).Account.ID, context.Query("link"))
	if issued != nil {
		context.Set("Cache-Control", "no-store")
		context.Set("Referrer-Policy", "no-referrer")
	}
	data, dataError := members.Users(held(context), context.Query("q"), members.ParsePage(context.Query("page")), editing, issued, roleID)
	if dataError != nil {
		return pageFailure(dataError)
	}
	var transferring uint
	fmt.Sscan(context.Query("transfer"), &transferring)
	if transferring != 0 {
		data.Transferring, dataError = members.OwnershipRecipient(held(context), transferring)
		if dataError != nil {
			return pageFailure(dataError)
		}
	}
	data.InvitationsOnly = context.Query("view") == "invitations"
	meta.SetPageTitle(context, data.Title)
	return shortcuts.Render(context, "administration/users", data)
}

func Roles(context fiber.Ctx) error {
	var editing uint
	fmt.Sscan(context.Query("edit"), &editing)
	data, dataError := members.Roles(held(context), editing, context.Query("new") == "1")
	if dataError != nil {
		return pageFailure(dataError)
	}
	data.MemberTab = context.Query("tab") == "users"
	data.DisplayTab = !data.MemberTab && context.Query("tab") != "permissions"
	data.ColourOptions = []string{"#1abc9c", "#2ecc71", "#3498db", "#9b59b6", "#e91e63", "#f1c40f", "#e67e22", "#e74c3c", "#95a5a6", "#607d8b", "#11806a", "#1f8b4c", "#206694", "#71368a", "#ad1457", "#c27c0e", "#a84300", "#992d22", "#979c9f", "#546e7a"}
	meta.SetPageTitle(context, data.Title)
	return shortcuts.Render(context, "administration/roles", data)
}

func SMTP(context fiber.Ctx) error {
	settings, settingsError := mailing.Settings()
	if settingsError != nil {
		return settingsError
	}
	password, passwordError := mailing.Password(settings)
	if passwordError != nil {
		return passwordError
	}
	context.Set("Cache-Control", "no-store")
	meta.SetPageTitle(context, "SMTP settings")
	return shortcuts.Render(context, "administration/smtp", collections.Record[string, any]{"Title": "SMTP settings", "SMTP": settings, "Configured": mailing.Configured(settings), "SMTPPassword": password, "SecurityOptions": collections.OptionsOf("starttls", "STARTTLS", "tls", "Implicit TLS")})
}

func Sharing(context fiber.Ctx) error {
	actor := held(context)
	record, recordError := access.Database(actor, context.Params("name"))
	if recordError != nil {
		return fiber.ErrNotFound
	}
	if record.OwnerAccountID != actor.Account.ID && !actor.Can(permissions.DatabaseShare) {
		return fiber.ErrForbidden
	}
	var users []models.Account
	if findError := database.DB.Where("active = ? AND id <> ?", true, record.OwnerAccountID).Order("username asc").Find(&users).Error; findError != nil {
		return findError
	}
	var shares []models.DatabaseShare
	if findError := database.DB.Where("database_id = ?", record.ID).Find(&shares).Error; findError != nil {
		return findError
	}
	choices := make([]members.Choice, 0, len(users))
	for _, user := range users {
		profile := auth.ToAccountView(&user)
		target, findError := access.Load(user.ID)
		if findError != nil {
			return findError
		}
		choice := members.Choice{ID: user.ID, Name: profile.DisplayName, Email: profile.Email, Picture: profile.ProfilePicture, Initial: profile.Initial, Implicit: target.All()}
		for _, share := range shares {
			if share.AccountID == user.ID {
				choice.Selected = true
			}
		}
		choices = append(choices, choice)
	}
	meta.SetPageTitle(context, "Database sharing")
	meta.SetSection(context, record.Name, "sharing")
	return shortcuts.Render(context, "administration/sharing", collections.Record[string, any]{"Title": "Database sharing", "Database": record.Name, "Members": choices})
}
