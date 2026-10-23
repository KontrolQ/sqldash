package databases

import (
	"context"
	"errors"
	"net/http"

	"sqldash/config"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/services/postgresconnections/overviewhelper"
	primarycredentials "sqldash/services/postgresconnections/primary"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func postgresContext(ctx context.Context, record *models.Database, shown *ShowContext) *fiber.Error {
	shown.Host, shown.Port = config.PostgresHost(), config.Postgres.PublicPort
	shown.Username = record.Name
	shown.Address = overviewhelper.Address(shown.Host, shown.Port)
	shown.URL = overviewhelper.ConnectionURL(shown.Host, shown.Port, record.Name, shown.Username)
	actor := access.FromContext(ctx)
	if !record.Available() || !actor.Can(permissions.PostgresCredentialsManage) || !actor.Can(permissions.PostgresDataRead) || !actor.Can(permissions.PostgresDataWrite) {
		return nil
	}
	credentials, credentialError := overviewhelper.Current(ctx, record)
	if errors.Is(credentialError, primarycredentials.Refused) {
		return shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	if credentialError != nil {
		shown.PasswordProblem = primarycredentials.Unavailable.Error()
		return nil
	}
	shown.PostgresCredentials = credentials
	shown.Username = credentials.Username
	shown.URL = credentials.URL
	return nil
}
