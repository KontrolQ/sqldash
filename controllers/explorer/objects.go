package explorer

import (
	"net/http"
	"net/url"

	"sqldash/services/audit"
	service "sqldash/services/explorer"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func ManageObject(context fiber.Ctx) error {
	asked, parseError := meta.Body[service.ObjectRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	asked.Schema = asked.SelectedSchema()
	name := context.Params(NameParameter)
	changed := false
	if changeError := service.ManageObject(context.Context(), name, asked); changeError != nil {
		if changeError.Code == http.StatusForbidden {
			return changeError
		}
		sessions.Complain(context, changeError.Message)
	} else {
		changed = true
		sessions.Report(context, "The database structure was updated.")
		object := asked.Name
		if object == "" {
			object = asked.Table
		}
		if asked.Kind == "column" {
			object = asked.Table + "." + asked.Column
		}
		audit.Note(context, name, audit.ObjectAction(asked.Action), asked.Action+" "+asked.Kind+" "+asked.Schema+"."+object)
	}
	if changed && asked.Kind == "schema" && asked.Action == "drop" {
		return shortcuts.RedirectToPath(context, ExplorePath+url.PathEscape(name)+"/structure?schema=pg_catalog")
	}
	query := url.Values{"schema": {asked.Schema}}
	relation := asked.Kind == "table" || asked.Kind == "view" || asked.Kind == "materialized view"
	if changed && asked.Kind == "schema" && asked.Action == "rename" {
		query.Set("schema", asked.NewName)
	}
	if asked.Table != "" && !(changed && asked.Action == "drop" && relation) {
		table := asked.Table
		if changed && asked.Action == "rename" && relation {
			table = asked.NewName
		}
		query.Set("table", table)
	}
	if changed && asked.Action == "create" && relation {
		query.Set("table", asked.Name)
	}
	return shortcuts.RedirectToPath(context, ExplorePath+url.PathEscape(name)+"/structure?"+query.Encode())
}

func CancelConsole(context fiber.Ctx) error {
	name := context.Params(NameParameter)
	if cancelError := service.CancelConsole(context.Context(), name, context.FormValue("run_id")); cancelError != nil {
		return cancelError
	}
	audit.Note(context, name, audit.StatementCancelled, "Cancellation requested for a running console submission")
	return context.SendStatus(http.StatusNoContent)
}
