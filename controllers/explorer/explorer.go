package explorer

import (
	"net/http"
	"net/url"

	service "sqldash/services/explorer"
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
		return backTo(context, name, asked.Back, ProblemParameter, saveError.Message)
	}

	return backTo(context, name, asked.Back, DoneParameter, service.CellSaved)
}

func DeleteRow(context fiber.Ctx) error {
	asked, parseError := meta.Body[RowRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if deleteError := service.DeleteRow(context.Context(), name, asked.Table, asked.Key); deleteError != nil {
		return backTo(context, name, asked.Back, ProblemParameter, deleteError.Message)
	}

	return backTo(context, name, asked.Back, DoneParameter, service.RowDeleted)
}

func RunConsole(context fiber.Ctx) error {
	asked, parseError := meta.Body[ConsoleRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	return shortcuts.RedirectToPath(
		context,
		ExplorePath+url.PathEscape(name)+ConsoleSuffix+"?statement="+url.QueryEscape(asked.Statement),
	)
}

func backTo(context fiber.Ctx, name string, back string, parameter string, message string) error {
	target := ExplorePath + url.PathEscape(name) + BrowseSuffix

	if back != "" {
		target += "?" + back + "&"
	} else {
		target += "?"
	}

	return shortcuts.RedirectToPath(context, target+parameter+"="+url.QueryEscape(message))
}
