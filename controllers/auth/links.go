package auth

import (
	"errors"

	"sqldash/services/audit"
	"sqldash/services/members"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/passwords"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type LinkRequest struct {
	Token       string `form:"token"`
	Username    string `form:"username"`
	DisplayName string `form:"display_name"`
	Password    string `form:"password"`
}

func linkFailure(failure error) string {
	for _, known := range []error{members.Invalid, members.LinkInvalid, members.Refused, members.Duplicate} {
		if errors.Is(failure, known) {
			return known.Error()
		}
	}
	if failure.Error() == passwords.TooShort {
		return passwords.TooShort
	}
	return "Your credentials could not be saved. Try again."
}

func AcceptInvitation(context fiber.Ctx) error {
	context.Set("Referrer-Policy", "no-referrer")
	context.Set("Cache-Control", "no-store")
	asked, parseError := meta.Body[LinkRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(fiber.StatusBadRequest, FormUnreadable)
	}
	account, acceptError := members.AcceptInvitation(asked.Token, asked.Username, asked.DisplayName, asked.Password)
	if acceptError != nil {
		return shortcuts.ServiceError(fiber.StatusBadRequest, linkFailure(acceptError))
	}
	audit.Record(account.Username, "", "invitation.accepted", "")
	sessions.Report(context, "Your account is ready. Sign in with your new credentials.")
	return shortcuts.Redirect(context, LoginRoute)
}

func RecoverAccount(context fiber.Ctx) error {
	context.Set("Referrer-Policy", "no-referrer")
	context.Set("Cache-Control", "no-store")
	asked, parseError := meta.Body[LinkRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(fiber.StatusBadRequest, FormUnreadable)
	}
	if recoverError := members.RecoverAccount(asked.Token, asked.Password); recoverError != nil {
		return shortcuts.ServiceError(fiber.StatusBadRequest, linkFailure(recoverError))
	}
	if forgetError := sessions.Forget(context); forgetError != nil {
		return forgetError
	}
	sessions.Report(context, "Your password was changed. Sign in again.")
	return shortcuts.Redirect(context, LoginRoute)
}
