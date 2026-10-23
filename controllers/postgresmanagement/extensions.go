package postgresmanagement

import (
	"net/http"
	"net/url"

	"sqldash/services/audit"
	service "sqldash/services/postgresmanagement"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type extensionRequest struct {
	Extension string `form:"extension"`
	Operation string `form:"operation"`
}

func ManageExtension(context fiber.Ctx) error {
	asked, err := meta.Body[extensionRequest](context)
	if err != nil || asked.Operation != "install" && asked.Operation != "remove" {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	name := context.Params(NameParameter)
	install := asked.Operation == "install"
	if record, err := service.ChangeExtension(context.Context(), name, asked.Extension, install); err != nil {
		sessions.Complain(context, err.Message)
	} else {
		action := service.ExtensionRemoved
		message := ExtensionRemoved
		if install {
			action = service.ExtensionInstalled
			message = ExtensionInstalled
		}
		sessions.Report(context, message)
		audit.RecordDatabase(audit.ActorOf(context), record, action, asked.Extension)
	}
	return shortcuts.RedirectToPath(context, DatabasePath+url.PathEscape(name)+"/extensions")
}
