package auth

import (
	"net/http"
	"net/url"

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
		return shortcuts.RedirectToPath(context, LoginPath+"?problem="+url.QueryEscape(signInError.Message))
	}

	sessions.Remember(context, held.ID)

	return shortcuts.Redirect(context, HomeRoute)
}

func SignOut(context fiber.Ctx) error {
	sessions.Forget(context)

	return shortcuts.Redirect(context, LoginRoute)
}
