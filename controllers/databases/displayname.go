package databases

import (
	"net/http"

	"sqldash/services/audit"
	service "sqldash/services/databases"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type DisplayNameRequest struct {
	DisplayName string `form:"displayName"`
}

func SaveDisplayName(context fiber.Ctx) error {
	name := context.Params(NameParameter)
	asked, parsingError := meta.Body[DisplayNameRequest](context)
	if parsingError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	if savingError := service.SaveDisplayName(context.Context(), name, asked.DisplayName); savingError != nil {
		sessions.Complain(context, savingError.Message)
	} else {
		sessions.Report(context, DisplayNameSaved)
		audit.Note(context, name, audit.SettingsSaved, DisplayNameAuditDetail)
	}
	return backTo(context, name)
}
