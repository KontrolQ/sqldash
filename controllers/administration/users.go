package administration

import (
	"github.com/gofiber/fiber/v3"
	"sqldash/services/audit"
	"sqldash/services/members"
	"sqldash/utils/meta"
)

type UserRequest struct {
	Email     string `form:"email"`
	Roles     []uint `form:"roles"`
	Databases []uint `form:"databases"`
	All       bool   `form:"all"`
	SendEmail bool   `form:"send_email"`
}

func IssueInvitation(context fiber.Ctx) error {
	asked, parseError := meta.Body[UserRequest](context)
	if parseError != nil {
		return complete(context, InviteCreated, members.Invalid, UsersPath)
	}
	issued, issueError := members.Issue(actor(context), members.Invitation, asked.Email, 0, nil, nil, false, asked.SendEmail)
	return showLink(context, issued, issueError, InviteCreated)
}
func IssueRecovery(context fiber.Ctx) error {
	issued, issueError := members.Issue(actor(context), members.Recovery, "", identifier(context), nil, nil, false)
	return showLink(context, issued, issueError, RecoveryCreated)
}
func showLink(context fiber.Ctx, issued *members.IssuedLink, issueError error, action string) error {
	if issueError != nil {
		return complete(context, action, issueError, UsersPath)
	}
	key, holdError := members.HoldIssuedLink(actor(context).Account.ID, issued)
	if holdError != nil {
		return complete(context, action, holdError, UsersPath)
	}
	audit.Note(context, "", action, "")
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "no-referrer")
	return context.Redirect().Status(fiber.StatusSeeOther).To(UsersPath + "?view=invitations&link=" + key)
}
func SaveUser(context fiber.Ctx) error {
	asked, parseError := meta.Body[UserRequest](context)
	if parseError != nil {
		return complete(context, Changed, members.Invalid, UsersPath)
	}
	return complete(context, UserAccessChanged, members.SaveUser(actor(context), identifier(context), asked.Roles, asked.Databases, asked.All), UsersPath)
}
func Activate(context fiber.Ctx) error {
	active := context.FormValue("active") == "true"
	action := UserDeactivated
	if active {
		action = UserActivated
	}
	return complete(context, action, members.Activate(actor(context), identifier(context), active), UsersPath)
}
func RevokeSessions(context fiber.Ctx) error {
	return complete(context, UserSessionsRevoked, members.RevokeSessions(actor(context), identifier(context)), UsersPath)
}
func CancelInvitation(context fiber.Ctx) error {
	return complete(context, InviteCancelled, members.CancelLink(actor(context), identifier(context)), UsersPath)
}
func ReissueInvitation(context fiber.Ctx) error {
	issued, issueError := members.ReissueLink(actor(context), identifier(context))
	return showLink(context, issued, issueError, InviteCreated)
}
