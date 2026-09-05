package explorer

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	service "sqldash/services/explorer"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveCells(context fiber.Ctx) error {
	asked, parseError := meta.Body[CellRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)
	edits := make([]service.CellEdit, 0, len(asked.Columns))

	for index, column := range asked.Columns {
		if index >= len(asked.Keys) || index >= len(asked.Values) {
			break
		}

		clear := index < len(asked.Clears) && asked.Clears[index] == CheckedValue

		edits = append(edits, service.CellEdit{
			Column: column, Key: asked.Keys[index], Value: asked.Values[index], Clear: clear,
		})
	}

	if saveError := service.UpdateCells(context.Context(), name, asked.Table, edits); saveError != nil {
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, savedMessage(len(edits)))
	}

	return backTo(context, name, asked.Back)
}

func DeleteRows(context fiber.Ctx) error {
	asked, parseError := meta.Body[RowRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if deleteError := service.DeleteRows(context.Context(), name, asked.Table, asked.Keys); deleteError != nil {
		sessions.Complain(context, deleteError.Message)
	} else {
		sessions.Report(context, fmt.Sprintf(service.RowsDeleted, len(asked.Keys)))
	}

	return backTo(context, name, asked.Back)
}

func RunConsole(context fiber.Ctx) error {
	asked, parseError := meta.Body[ConsoleRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	data, dataError := service.RunConsole(context.Context(), name, asked.Statement)
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, ConsoleTemplate, data)
}

func savedMessage(count int) string {
	if count == 1 {
		return service.OneCellSaved
	}

	return fmt.Sprintf(service.CellsSaved, count)
}

func backTo(context fiber.Ctx, name string, back string) error {
	target := ExplorePath + url.PathEscape(name) + BrowseSuffix

	if back != "" {
		target += "?" + back
	}

	return shortcuts.RedirectToPath(context, target)
}

func InsertRow(context fiber.Ctx) error {
	asked, parseError := meta.Body[InsertRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)
	values := make(map[string]string)

	for _, column := range strings.Split(asked.Columns, ColumnSeparator) {
		column = strings.TrimSpace(column)

		if column != "" {
			values[column] = context.FormValue(ColumnPrefix + column)
		}
	}

	if insertError := service.InsertRow(context.Context(), name, asked.Table, values); insertError != nil {
		sessions.Complain(context, insertError.Message)
	} else {
		sessions.Report(context, service.RowAdded)
	}

	return backTo(context, name, asked.Back)
}
