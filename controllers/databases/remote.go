package databases

import (
	"io"
	"net/http"

	pages "sqldash/pages/databases"
	"sqldash/pgtransfer"
	"sqldash/services/audit"
	service "sqldash/services/databases"
	"sqldash/utils/collections"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type RemoteCloneRequest struct {
	Name     string `form:"name"`
	Host     string `form:"source_host"`
	Port     int    `form:"source_port"`
	Database string `form:"source_database"`
	Username string `form:"source_username"`
	Password string `form:"source_password"`
	SSLMode  string `form:"source_sslmode"`
}

func CloneRemote(context fiber.Ctx) error {
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "no-referrer")
	asked, parseError := meta.Body[RemoteCloneRequest](context)
	if parseError != nil {
		return remoteCloneFailure(context, http.StatusBadRequest, FormUnreadable)
	}
	asked.Name = service.NormalizeName(asked.Name)
	source := pgtransfer.RemoteSource{Host: asked.Host, Port: asked.Port, Database: asked.Database, Username: asked.Username, Password: asked.Password, SSLMode: asked.SSLMode}
	defer func() { asked.Password, source.Password = "", "" }()
	if header, err := context.FormFile("source_ca"); err == nil {
		handle, err := header.Open()
		if err != nil {
			return remoteCloneFailure(context, http.StatusBadRequest, service.RemoteRefused)
		}
		certificate, readError := io.ReadAll(io.LimitReader(handle, (1<<20)+1))
		handle.Close()
		if readError != nil || len(certificate) > 1<<20 {
			return remoteCloneFailure(context, http.StatusBadRequest, service.RemoteRefused)
		}
		source.RootCertificate = certificate
	}
	cloneError := service.CloneRemote(context.Context(), asked.Name, source)
	asked.Password, source.Password = "", ""
	if cloneError != nil {
		return remoteCloneFailure(context, cloneError.Code, cloneError.Message)
	}
	audit.Note(context, asked.Name, audit.DatabaseCloned, "Copied from a remote PostgreSQL database")
	if context.Get("X-Remote-Clone") == "1" {
		return context.JSON(collections.Record[string, any]{"ok": true})
	}
	return shortcuts.Redirect(context, IndexRoute)
}

func remoteCloneFailure(context fiber.Ctx, status int, message string) error {
	context.Status(status)
	if context.Get("X-Remote-Clone") == "1" {
		return context.JSON(collections.Record[string, any]{"error": message})
	}
	return pages.RemoteCloneFailure(context, &service.RemoteCloneForm{
		Name:     context.FormValue("name"),
		Host:     context.FormValue("source_host"),
		Port:     context.FormValue("source_port"),
		Database: context.FormValue("source_database"),
		Username: context.FormValue("source_username"),
		SSLMode:  context.FormValue("source_sslmode"),
		Error:    message,
	})
}
