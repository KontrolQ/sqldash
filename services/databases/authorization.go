package databases

import (
	"context"
	"net/http"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func FreshActor(ctx context.Context) (context.Context, *access.AccountAccess, *fiber.Error) {
	actor, err := access.Load(access.AccountID(ctx))
	if err != nil || actor == nil || !actor.Account.Active {
		return ctx, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	return access.WithContext(ctx, actor), actor, nil
}

func Authorize(ctx context.Context, name string, permission string, ready bool) (context.Context, *models.Database, *fiber.Error) {
	fresh, actor, err := FreshActor(ctx)
	if err != nil {
		return ctx, nil, err
	}
	record, findError := access.Database(actor, name)
	if findError != nil {
		return fresh, nil, shortcuts.ServiceError(http.StatusNotFound, DatabaseMissing)
	}
	if !actor.CanDatabase(record, permission) {
		return fresh, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	if record.Kind() != models.EngineLibSQL && record.Kind() != models.EnginePostgres {
		return fresh, nil, shortcuts.ServiceError(http.StatusBadRequest, EngineRefused)
	}
	if ready && !record.Available() {
		return fresh, nil, shortcuts.ServiceError(http.StatusConflict, DatabaseNotReady)
	}
	return engines.WithDatabase(fresh, record), record, nil
}

func creationActor(ctx context.Context, engine string, permission string) (context.Context, *access.AccountAccess, *fiber.Error) {
	fresh, actor, err := FreshActor(ctx)
	if err != nil {
		return fresh, nil, err
	}
	if engine != models.EngineLibSQL && engine != models.EnginePostgres {
		return fresh, nil, shortcuts.ServiceError(http.StatusBadRequest, EngineRefused)
	}
	if engine == models.EnginePostgres && !config.Postgres.Enabled {
		return fresh, nil, shortcuts.ServiceError(http.StatusServiceUnavailable, PostgresUnavailable)
	}
	if !actor.Can(permissions.ForEngine(permission, engine)) {
		return fresh, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	if engine == models.EnginePostgres && (!actor.Can(permissions.PostgresCredentialsManage) || !actor.Can(permissions.PostgresDataRead) || !actor.Can(permissions.PostgresDataWrite)) {
		return fresh, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	return fresh, actor, nil
}

func selectedEngine(engine string) string {
	if engine == "" {
		return models.EngineLibSQL
	}
	return engine
}
