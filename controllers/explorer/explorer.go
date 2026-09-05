package explorer

import (
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
		audit.Note(context, name, audit.RowsChanged, countedAs(len(edits), OneChangedFormat, ChangedFormat, asked.Table))
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
		sessions.Report(context, deletedMessage(len(asked.Keys)))
		audit.Note(context, name, audit.RowsDeleted, countedAs(len(asked.Keys), OneDeletedFormat, DeletedFormat, asked.Table))
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

	audit.Note(context, name, audit.StatementRun, shortened(asked.Statement))

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

	if saveError := service.SaveSnippet(name, asked.Snippet, asked.Name, asked.Statement); saveError != nil {
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, service.SnippetSaved)
	}

	return shortcuts.RedirectToPath(context, ExplorePath+url.PathEscape(name)+ConsoleSuffix)
}

func RemoveSnippet(context fiber.Ctx) error {
	asked, parseError := meta.Body[SnippetRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if removeError := service.RemoveSnippet(name, asked.Snippet); removeError != nil {
		sessions.Complain(context, removeError.Message)
	} else {
		sessions.Report(context, service.SnippetRemoved)
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

	if insertError := service.InsertRow(context.Context(), name, asked.Table, values); insertError != nil {
		sessions.Complain(context, insertError.Message)
	} else {
		sessions.Report(context, service.RowAdded)
		audit.Note(context, name, audit.RowInserted, fmt.Sprintf(InsertedFormat, asked.Table))
	}

	return backTo(context, name, asked.Back)
}

func shortened(statement string) string {
	trimmed := strings.Join(strings.Fields(statement), " ")

	if len(trimmed) > StatementLength {
		return trimmed[:StatementLength] + Ellipsis
	}

	return trimmed
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
