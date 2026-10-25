package databases

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"sqldash/models"
	"sqldash/permissions"
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

	asked.Name = service.NormalizeName(asked.Name)
	importError := service.CreateFromUploadEngine(context.Context(), asked.Name, asked.Engine, header.Filename, handle)
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

	asked.Name = service.NormalizeName(asked.Name)
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
	_, record, err := service.Authorize(context.Context(), name, permissions.DatabaseExport, true)
	if err != nil {
		return err
	}
	if record.Kind() == models.EnginePostgres {
		return ExportPostgresSQL(context)
	}
	if record.Kind() == models.EngineRedis {
		return ExportRedis(context)
	}

	context.Set(DispositionHeader, `attachment; filename="`+name+`.sql"`)
	context.Set(ContentTypeHeader, DumpContentType)

	if writeError := service.WriteDump(context.Context(), name, context.Response().BodyWriter()); writeError != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, service.ExportRefused)
	}

	audit.Note(context, name, audit.DatabaseExported, DumpKind)

	return nil
}

func ExportPostgresArchive(context fiber.Ctx) error {
	return exportPostgres(context, "custom", ".dump")
}
func ExportPostgresSQL(context fiber.Ctx) error { return exportPostgres(context, "sql", ".sql") }

func exportPostgres(context fiber.Ctx, format, suffix string) error {
	name := context.Params(NameParameter)
	_, record, err := service.Authorize(context.Context(), name, permissions.DatabaseExport, true)
	if err != nil {
		return err
	}
	if record.Kind() != models.EnginePostgres {
		return shortcuts.ServiceError(http.StatusBadRequest, service.EngineRefused)
	}
	held, exportError := service.PostgresExport(context.Context(), record.Name, format)
	if exportError != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, service.ExportRefused)
	}
	context.Set(DispositionHeader, `attachment; filename="`+record.Name+suffix+`"`)
	context.Set(ContentTypeHeader, "application/octet-stream")
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "no-referrer")
	audit.NoteDatabase(context, record, audit.DatabaseExported, "PostgreSQL "+format+" dump generated")
	return context.SendStream(held)
}
