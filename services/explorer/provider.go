package explorer

import (
	"context"
	"net/http"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func authorizedDatabase(requestContext context.Context, databaseName string, permission string) (*models.Database, *fiber.Error) {
	record, _, permissionError := authorizedDatabaseAccess(requestContext, databaseName, permission)
	return record, permissionError
}

func authorizedDatabaseAccess(requestContext context.Context, databaseName string, permission string) (*models.Database, *access.AccountAccess, *fiber.Error) {
	actor := access.FromContext(requestContext)
	if actor == nil {
		return nil, nil, shortcuts.ServiceError(http.StatusForbidden, "Database access denied.")
	}
	fresh, loadError := access.Load(actor.Account.ID)
	if loadError != nil || !fresh.Account.Active {
		return nil, nil, shortcuts.ServiceError(http.StatusForbidden, "Database access denied.")
	}
	record, findError := access.Database(fresh, databaseName)
	if findError != nil || !fresh.CanDatabase(record, permission) {
		return nil, nil, shortcuts.ServiceError(http.StatusForbidden, "Database access denied.")
	}
	resolved, resolveError := engines.Resolve(requestContext, databaseName)
	if resolveError != nil || resolved.ID != record.ID || !record.Available() {
		return nil, nil, shortcuts.ServiceError(http.StatusServiceUnavailable, "The database is not ready.")
	}
	return record, fresh, nil
}

func catalogFor(record *models.Database) (engines.Catalog, *fiber.Error) {
	backend, backendError := engines.For(record)
	if backendError != nil {
		return nil, shortcuts.ServiceError(http.StatusServiceUnavailable, "The database engine is unavailable.")
	}
	catalog, supported := backend.(engines.Catalog)
	if !supported {
		return nil, shortcuts.ServiceError(http.StatusNotImplemented, "This engine does not provide a catalog.")
	}
	return catalog, nil
}

func selectedSchema(schemas []string) string {
	if len(schemas) > 0 && schemas[0] != "" {
		return schemas[0]
	}
	return "public"
}

func databaseContext(parent context.Context, record *models.Database, mode engines.Mode) context.Context {
	return engines.WithMode(engines.WithDatabase(parent, record), mode)
}

func CanWrite(requestContext context.Context, name string) bool {
	record, permissionError := authorizedDatabase(requestContext, name, permissions.DataWrite)
	return permissionError == nil && record != nil
}
