package databases

import (
	"fmt"
	"net/http"

	"sqldash/permissions"
	"sqldash/services/audit"
	service "sqldash/services/databases"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Create(context fiber.Ctx) error {
	asked, parseError := meta.Body[CreateRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	record, createError := service.CreateEngineRecord(context.Context(), asked.Name, asked.Engine)
	if createError != nil {
		sessions.Complain(context, createError.Message)
		return shortcuts.Redirect(context, IndexRoute)
	}

	audit.NoteDatabase(context, record, audit.DatabaseCreated, StartedEmpty)
	sessions.Report(context, fmt.Sprintf(CreatedWithName, record.Label()))

	return shortcuts.Redirect(context, IndexRoute)
}

func Delete(context fiber.Ctx) error {
	name := context.Params(NameParameter)
	_, record, guardError := service.Authorize(context.Context(), name, permissions.DatabaseDelete, false)
	if guardError != nil {
		sessions.Complain(context, guardError.Message)
		return shortcuts.Redirect(context, IndexRoute)
	}

	if deleteError := service.Delete(context.Context(), name); deleteError != nil {
		sessions.Complain(context, deleteError.Message)
		return shortcuts.Redirect(context, IndexRoute)
	}

	audit.RecordDatabase(audit.ActorOf(context), record, audit.DatabaseDeleted, "")

	return shortcuts.Redirect(context, IndexRoute)
}
