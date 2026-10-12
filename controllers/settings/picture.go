package settings

import (
	"sqldash/services/audit"
	service "sqldash/services/settings"
	"sqldash/sessions"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SavePicture(context fiber.Ctx) error {
	header, fileError := context.FormFile(PictureField)
	if fileError != nil {
		sessions.Complain(context, PictureMissing)
		return shortcuts.RedirectToPath(context, IndexPath)
	}

	handle, openError := header.Open()
	if openError != nil {
		sessions.Complain(context, PictureMissing)
		return shortcuts.RedirectToPath(context, IndexPath)
	}
	defer handle.Close()

	if saveError := service.SavePicture(sessions.Remembered(context), handle); saveError != nil {
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, service.PictureSaved)
		audit.Note(context, "", audit.AccountChanged, PictureAuditDetail)
	}

	return shortcuts.RedirectToPath(context, IndexPath)
}
