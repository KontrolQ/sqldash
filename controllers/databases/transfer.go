package databases

import (
	"net/http"
	"net/url"
	"time"

	service "sqldash/services/databases"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Import(context fiber.Ctx) error {
	asked, parseError := meta.Body[ImportRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	header, fileError := context.FormFile(DumpField)
	if fileError != nil {
		return shortcuts.RedirectToPath(context, IndexPath+"?problem="+url.QueryEscape(FileMissing))
	}

	handle, openError := header.Open()
	if openError != nil {
		return shortcuts.RedirectToPath(context, IndexPath+"?problem="+url.QueryEscape(FileMissing))
	}

	defer handle.Close()

	importError := service.CreateFromUpload(context.Context(), asked.Name, header.Filename, handle)
	if importError != nil {
		return shortcuts.RedirectToPath(context, IndexPath+"?problem="+url.QueryEscape(importError.Message))
	}

	return shortcuts.Redirect(context, IndexRoute)
}

func Fork(context fiber.Ctx) error {
	asked, parseError := meta.Body[ForkRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	var at *time.Time

	if asked.Moment != "" {
		parsed, momentError := time.ParseInLocation(service.MomentLayout, asked.Moment, time.Local)
		if momentError != nil {
			return backTo(context, name, ProblemParameter, service.MomentUnusable)
		}

		at = &parsed
	}

	if forkError := service.Fork(context.Context(), name, asked.Name, at); forkError != nil {
		return backTo(context, name, ProblemParameter, forkError.Message)
	}

	return shortcuts.RedirectToPath(context, ShowPath+url.PathEscape(asked.Name))
}

func Export(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	context.Set(DispositionHeader, `attachment; filename="`+name+`.sql"`)
	context.Set(ContentTypeHeader, DumpContentType)

	if writeError := service.WriteDump(context.Context(), name, context.Response().BodyWriter()); writeError != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, service.ExportRefused)
	}

	return nil
}
