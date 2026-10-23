package postgresmanagement

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"sqldash/services/audit"
	service "sqldash/services/postgresmanagement"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type activityRequest struct {
	PID          uint32 `form:"pid"`
	BackendStart string `form:"backend_start"`
}

func CancelActivity(context fiber.Ctx) error    { return changeActivity(context, false) }
func TerminateActivity(context fiber.Ctx) error { return changeActivity(context, true) }

func changeActivity(context fiber.Ctx, terminate bool) error {
	asked, err := meta.Body[activityRequest](context)
	if err != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	startedAt, parseError := time.Parse(time.RFC3339Nano, asked.BackendStart)
	if parseError != nil || asked.PID == 0 {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	name := context.Params(NameParameter)
	if record, err := service.ChangeActivity(context.Context(), name, service.SessionTarget{PID: asked.PID, BackendStart: startedAt}, terminate); err != nil {
		sessions.Complain(context, err.Message)
	} else {
		action := service.ActivityCancelled
		message := QueryCancelled
		if terminate {
			action = service.ActivityTerminated
			message = ConnectionTerminated
		}
		sessions.Report(context, message)
		audit.RecordDatabase(audit.ActorOf(context), record, action, fmt.Sprintf("Connection %d", asked.PID))
	}
	return shortcuts.RedirectToPath(context, DatabasePath+url.PathEscape(name)+"/activity")
}
