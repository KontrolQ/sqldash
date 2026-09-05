package auth

import (
	"net/http"

	"sqldash/services/audit"
	service "sqldash/services/auth"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SignIn(context fiber.Ctx) error {
	asked, parseError := meta.Body[SignInRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	held, signInError := service.SignIn(asked.Username, asked.Password)
	if signInError != nil {
		sessions.Complain(context, signInError.Message)
		return shortcuts.Redirect(context, LoginRoute)
	}

	wanted := sessions.Intended(context)

	sessions.Remember(context, held.ID)
	audit.Record(held.Username, "", audit.SignedIn, "")

	if wanted != "" {
		return shortcuts.RedirectToPath(context, wanted)
	}

	return shortcuts.Redirect(context, HomeRoute)
}

func SignOut(context fiber.Ctx) error {
	audit.Note(context, "", audit.SignedOut, "")
	sessions.Forget(context)

	return shortcuts.Redirect(context, LoginRoute)
}
