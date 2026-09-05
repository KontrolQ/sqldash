package settings

import (
	"net/http"

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
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, service.DetailsSaved)
	}

	return shortcuts.RedirectToPath(context, IndexPath)
}

func ChangePassword(context fiber.Ctx) error {
	asked, parseError := meta.Body[PasswordRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	if changeError := service.ChangePassword(sessions.Remembered(context), asked.Current, asked.Wanted); changeError != nil {
		sessions.Complain(context, changeError.Message)
	} else {
		sessions.Report(context, service.PasswordSaved)
	}

	return shortcuts.RedirectToPath(context, IndexPath)
}
