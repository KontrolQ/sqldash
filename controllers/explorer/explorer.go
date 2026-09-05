package explorer

import (
	"net/http"
	"net/url"
	"strings"

	service "sqldash/services/explorer"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveCell(context fiber.Ctx) error {
	asked, parseError := meta.Body[CellRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	saveError := service.UpdateCell(
		context.Context(), name, asked.Table, asked.Column, asked.Key, asked.Value, asked.Clear == CheckedValue,
	)

	if saveError != nil {
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, service.CellSaved)
	}

	return backTo(context, name, asked.Back)
}

func DeleteRow(context fiber.Ctx) error {
	asked, parseError := meta.Body[RowRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if deleteError := service.DeleteRow(context.Context(), name, asked.Table, asked.Key); deleteError != nil {
		sessions.Complain(context, deleteError.Message)
	} else {
		sessions.Report(context, service.RowDeleted)
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
