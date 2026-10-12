package settings

import (
	"sqldash/services/audit"
	service "sqldash/services/settings"
	"sqldash/sessions"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func UseGravatar(context fiber.Ctx) error {
	if importError := service.UseGravatar(context.Context(), sessions.Remembered(context)); importError != nil {
		sessions.Complain(context, importError.Message)
	} else {
		sessions.Report(context, service.PictureSaved)
		audit.Note(context, "", audit.AccountChanged, PictureAuditDetail)
	}
	return shortcuts.RedirectToPath(context, IndexPath)
}
