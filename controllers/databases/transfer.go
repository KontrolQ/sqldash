package databases

import (
	"net/http"
	"net/url"
	"time"

	service "sqldash/services/databases"
	"sqldash/sessions"
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
		sessions.Complain(context, FileMissing)
		return shortcuts.Redirect(context, IndexRoute)
	}

	handle, openError := header.Open()
	if openError != nil {
		sessions.Complain(context, FileMissing)
		return shortcuts.Redirect(context, IndexRoute)
	}

	defer handle.Close()

	importError := service.CreateFromUpload(context.Context(), asked.Name, header.Filename, handle)
	if importError != nil {
		sessions.Complain(context, importError.Message)
		return shortcuts.Redirect(context, IndexRoute)
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
			sessions.Complain(context, service.MomentUnusable)
			return backTo(context, name)
		}

		at = &parsed
	}

	if forkError := service.Fork(context.Context(), name, asked.Name, at); forkError != nil {
		sessions.Complain(context, forkError.Message)
		return backTo(context, name)
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
