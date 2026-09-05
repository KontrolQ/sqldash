package databases

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"sqldash/services/audit"
	service "sqldash/services/databases"
	"sqldash/sessions"
	"sqldash/utils/logger"
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

	audit.Note(context, asked.Name, audit.DatabaseImported, fmt.Sprintf(LoadedFrom, header.Filename))

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

	detail := fmt.Sprintf(CopiedFrom, name)
	if at != nil {
		detail = fmt.Sprintf(CopiedAsAtWhen, name, asked.Moment)
	}

	audit.Note(context, asked.Name, audit.DatabaseForked, detail)

	return shortcuts.RedirectToPath(context, ShowPath+url.PathEscape(asked.Name))
}

func ExportFile(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	context.Set(DispositionHeader, `attachment; filename="`+name+`.db"`)
	context.Set(ContentTypeHeader, FileContentType)

	if writeError := service.WriteFile(context.Context(), name, context.Response().BodyWriter()); writeError != nil {
		logger.Errorf(LogPrefix, service.ExportFailedLog, name, writeError)
		return shortcuts.ServiceError(http.StatusBadGateway, service.ExportRefused)
	}

	audit.Note(context, name, audit.DatabaseExported, SQLiteFileKind)

	return nil
}

func Export(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	context.Set(DispositionHeader, `attachment; filename="`+name+`.sql"`)
	context.Set(ContentTypeHeader, DumpContentType)

	if writeError := service.WriteDump(context.Context(), name, context.Response().BodyWriter()); writeError != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, service.ExportRefused)
	}

	audit.Note(context, name, audit.DatabaseExported, DumpKind)

	return nil
}
