package postgresconnections

import (
	"context"
	"errors"
	"net/http"
	"time"

	"sqldash/config"
	"sqldash/models"
	"sqldash/services/access"
	"sqldash/services/postgresconnections/overviewhelper"
	primarycredentials "sqldash/services/postgresconnections/primary"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func ConnectionURL(host string, port int, databaseName, username string) string {
	return overviewhelper.ConnectionURL(host, port, databaseName, username)
}

func PasswordURL(host string, port int, databaseName, username, password string) string {
	return overviewhelper.PasswordURL(host, port, databaseName, username, password)
}

func issuedPassword(record *models.Database, connection *models.PostgresConnection, password string) *IssuedPassword {
	return &IssuedPassword{DatabaseID: record.ID, ConnectionID: connection.ID, Generation: connection.Generation, Label: connection.Label, Username: connection.Username, Password: password, Host: config.PostgresHost(), Port: config.Postgres.PublicPort, Database: record.Name, URL: PasswordURL(config.PostgresHost(), config.Postgres.PublicPort, record.Name, connection.Username, password)}
}

func GetIndexData(ctx context.Context, name string) (*IndexContext, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, models.ScopeReadWrite)
	if err != nil {
		return nil, err
	}
	shown := &IndexContext{Title: "Credentials", Name: name, Engine: models.EnginePostgres, EngineLabel: "PostgreSQL", Actions: access.Actions(access.FromContext(fresh), record), Host: config.PostgresHost(), Port: config.Postgres.PublicPort, Database: record.Name, DatabaseID: record.ID, Username: record.Name}
	credentials, credentialError := overviewhelper.Current(fresh, record)
	if errors.Is(credentialError, primarycredentials.Refused) {
		return nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	if credentialError != nil {
		shown.PasswordProblem = primarycredentials.Unavailable.Error()
		return shown, nil
	}
	shown.PrimaryIdentifier = credentials.Identifier
	shown.Username = credentials.Username
	shown.CredentialURL = credentials.URL
	shown.PasswordAvailable = true
	return shown, nil
}

func ValidateIssued(ctx context.Context, name string, result *IssuedPassword) *IssuedPassword {
	if result == nil {
		return nil
	}
	fresh, record, err := authorize(ctx, name, models.ScopeReadWrite)
	if err != nil || record.ID != result.DatabaseID {
		return nil
	}
	connection, err := findConnection(fresh, record.ID, result.ConnectionID)
	if err != nil || connection.Generation != result.Generation || !connection.Usable(time.Now()) || connection.AccountID != access.AccountID(fresh) {
		return nil
	}
	return result
}
