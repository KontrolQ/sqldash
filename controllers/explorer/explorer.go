package explorer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"sqldash/services/audit"
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

	if len(asked.Columns) != len(asked.Keys) || len(asked.Columns) != len(asked.Values) || (len(asked.States) > 0 && len(asked.States) != len(asked.Columns)) {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	if asked.ValueEncoding != "" && asked.ValueEncoding != "json" {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}
	if asked.ValueEncoding == "json" {
		for index, encoded := range asked.Values {
			var value *string
			if json.Unmarshal([]byte(encoded), &value) != nil || value == nil {
				return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
			}
			asked.Values[index] = *value
		}
	}
	name := context.Params(NameParameter)
	edits := make([]service.CellEdit, 0, len(asked.Columns))

	for index, column := range asked.Columns {
		if index >= len(asked.Keys) || index >= len(asked.Values) {
			break
		}

		clear := index < len(asked.Clears) && asked.Clears[index] == CheckedValue
		state := ""
		if index < len(asked.States) {
			state = asked.States[index]
		}
		if state == "keep" {
			continue
		}

		edits = append(edits, service.CellEdit{
			Column: column, Key: asked.Keys[index], Value: asked.Values[index], Clear: clear, State: state,
		})
	}

	if saveError := service.UpdateCells(context.Context(), name, asked.Table, edits, asked.Schema); saveError != nil {
		if saveError.Code == http.StatusConflict || saveError.Code == http.StatusForbidden {
			return saveError
		}
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, savedMessage(len(edits)))
		audit.Note(context, name, audit.RowsChanged, countedAs(len(edits), OneChangedFormat, ChangedFormat, relationName(asked.Schema, asked.Table)))
	}

	return backTo(context, name, asked.Back)
}

func DeleteRows(context fiber.Ctx) error {
	asked, parseError := meta.Body[RowRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if deleteError := service.DeleteRows(context.Context(), name, asked.Table, asked.Keys, asked.Schema); deleteError != nil {
		if deleteError.Code == http.StatusConflict || deleteError.Code == http.StatusForbidden {
			return deleteError
		}
		sessions.Complain(context, deleteError.Message)
	} else {
		sessions.Report(context, deletedMessage(len(asked.Keys)))
		audit.Note(context, name, audit.RowsDeleted, countedAs(len(asked.Keys), OneDeletedFormat, DeletedFormat, relationName(asked.Schema, asked.Table)))
	}

	return backTo(context, name, asked.Back)
}

func RunConsole(context fiber.Ctx) error {
	asked, parseError := meta.Body[ConsoleRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	data, dataError := service.RunConsole(context.Context(), name, asked.Statement, asked.Schema, asked.RunID)
	if dataError != nil {
		return dataError
	}

	detail := "SQL console submission"
	if data.Engine == "postgres" {
		detail = "PostgreSQL console submission"
	}
	if data.Failure != "" {
		detail += "; execution failed"
	}
	audit.Note(context, name, audit.StatementRun, detail)

	data.SnippetID = asked.Snippet

	for _, one := range data.Snippets {
		if one.ID == data.SnippetID {
			data.Name = one.Name
		}
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, ConsoleTemplate, data)
}

func SaveSnippet(context fiber.Ctx) error {
	asked, parseError := meta.Body[SnippetRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if permissionError := service.ConsoleAccess(context.Context(), name); permissionError != nil {
		return permissionError
	}

	if saveError := service.SaveSnippet(name, asked.Snippet, asked.Name, asked.Statement); saveError != nil {
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, service.SnippetSaved)
		audit.Note(context, name, audit.SnippetSaved, "")
	}

	return shortcuts.RedirectToPath(context, ExplorePath+url.PathEscape(name)+ConsoleSuffix)
}

func RemoveSnippet(context fiber.Ctx) error {
	asked, parseError := meta.Body[SnippetRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if permissionError := service.ConsoleAccess(context.Context(), name); permissionError != nil {
		return permissionError
	}

	if removeError := service.RemoveSnippet(name, asked.Snippet); removeError != nil {
		sessions.Complain(context, removeError.Message)
	} else {
		sessions.Report(context, service.SnippetRemoved)
		audit.Note(context, name, audit.SnippetRemoved, "")
	}

	return shortcuts.RedirectToPath(context, ExplorePath+url.PathEscape(name)+ConsoleSuffix)
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

	fields := make(map[string]service.FieldValue)
	for index, column := range asked.Names {
		if index >= len(asked.States) || index >= len(asked.Values) {
			return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
		}
		if _, duplicate := fields[column]; duplicate {
			return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
		}
		fields[column] = service.FieldValue{State: asked.States[index], Value: asked.Values[index]}
	}
	var insertError *fiber.Error
	if len(asked.Names) > 0 || asked.Schema != "" {
		insertError = service.InsertFields(context.Context(), name, asked.Schema, asked.Table, fields)
	} else {
		insertError = service.InsertRow(context.Context(), name, asked.Table, values)
	}
	if insertError != nil {
		sessions.Complain(context, insertError.Message)
	} else {
		sessions.Report(context, service.RowAdded)
		audit.Note(context, name, audit.RowInserted, fmt.Sprintf(InsertedFormat, relationName(asked.Schema, asked.Table)))
	}

	return backTo(context, name, asked.Back)
}

func relationName(schema string, table string) string {
	if schema == "" {
		return table
	}
	return schema + "." + table
}

func countedAs(count int, one string, many string, table string) string {
	if count == 1 {
		return fmt.Sprintf(one, table)
	}

	return fmt.Sprintf(many, count, table)
}

func deletedMessage(count int) string {
	if count == 1 {
		return service.OneRowDeleted
	}

	return fmt.Sprintf(service.RowsDeleted, count)
}
