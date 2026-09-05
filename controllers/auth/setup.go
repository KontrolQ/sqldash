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

func CreateAccount(context fiber.Ctx) error {
	asked, parseError := meta.Body[SetupRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	held, createError := service.CreateFirstAccount(asked.Username, asked.Email, asked.Password)
	if createError != nil {
		return shortcuts.RedirectToPath(context, SetupPath+"?problem="+url.QueryEscape(createError.Message))
	}

	sessions.Remember(context, held.ID)

	return shortcuts.Redirect(context, HomeRoute)
}
