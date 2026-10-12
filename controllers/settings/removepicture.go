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

func ClearPicture(context fiber.Ctx) error {
	if clearError := service.ClearPicture(sessions.Remembered(context)); clearError != nil {
		sessions.Complain(context, clearError.Message)
	} else {
		sessions.Report(context, service.PictureCleared)
		audit.Note(context, "", audit.AccountChanged, PictureClearedAuditDetail)
	}
	return shortcuts.RedirectToPath(context, IndexPath)
}

func DeletePicture(context fiber.Ctx) error {
	asked, parseError := meta.Body[PictureSelectionRequest](context)
	if parseError != nil || asked.PictureID == 0 {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	if deleteError := service.DeletePicture(sessions.Remembered(context), asked.PictureID); deleteError != nil {
		sessions.Complain(context, deleteError.Message)
	} else {
		sessions.Report(context, service.PictureDeleted)
		audit.Note(context, "", audit.AccountChanged, PictureDeletedAuditDetail)
	}
	return shortcuts.RedirectToPath(context, IndexPath)
}
