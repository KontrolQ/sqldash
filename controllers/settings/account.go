package settings

import (
	"net/http"
	"net/url"

	service "sqldash/services/settings"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveDetails(context fiber.Ctx) error {
	asked, parseError := meta.Body[DetailsRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	if saveError := service.SaveDetails(sessions.Remembered(context), asked.Username, asked.Email); saveError != nil {
		return back(context, ProblemParameter, saveError.Message)
	}

	return back(context, DoneParameter, service.DetailsSaved)
}

func ChangePassword(context fiber.Ctx) error {
	asked, parseError := meta.Body[PasswordRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	if changeError := service.ChangePassword(sessions.Remembered(context), asked.Current, asked.Wanted); changeError != nil {
		return back(context, ProblemParameter, changeError.Message)
	}

	return back(context, DoneParameter, service.PasswordSaved)
}

func back(context fiber.Ctx, parameter string, message string) error {
	return shortcuts.RedirectToPath(context, IndexPath+"?"+parameter+"="+url.QueryEscape(message))
}
