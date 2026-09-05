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

func Create(context fiber.Ctx) error {
	asked, parseError := meta.Body[CreateRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	if createError := service.Create(context.Context(), asked.Name); createError != nil {
		sessions.Complain(context, createError.Message)
		return shortcuts.Redirect(context, IndexRoute)
	}

	audit.Note(context, asked.Name, audit.DatabaseCreated, StartedEmpty)

	return shortcuts.Redirect(context, IndexRoute)
}

func Delete(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	if deleteError := service.Delete(context.Context(), name); deleteError != nil {
		sessions.Complain(context, deleteError.Message)
		return shortcuts.Redirect(context, IndexRoute)
	}

	audit.Note(context, name, audit.DatabaseDeleted, "")

	return shortcuts.Redirect(context, IndexRoute)
}
