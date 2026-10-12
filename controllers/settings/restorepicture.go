package settings

import (
	"net/http"

	"sqldash/services/audit"
	service "sqldash/services/settings"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func RestorePicture(context fiber.Ctx) error {
	asked, parseError := meta.Body[PictureSelectionRequest](context)
	if parseError != nil || asked.PictureID == 0 {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	if restoreError := service.RestorePicture(sessions.Remembered(context), asked.PictureID); restoreError != nil {
		sessions.Complain(context, restoreError.Message)
	} else {
		sessions.Report(context, service.PictureSaved)
		audit.Note(context, "", audit.AccountChanged, PictureAuditDetail)
	}
	return shortcuts.RedirectToPath(context, IndexPath)
}
