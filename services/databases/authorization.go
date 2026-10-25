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
	fresh, actor, err := access.Refresh(ctx)
	if err != nil {
		return ctx, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	return fresh, actor, nil
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
	if !models.IsKnownEngine(record.Kind()) {
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
	if !models.IsKnownEngine(engine) {
		return fresh, nil, shortcuts.ServiceError(http.StatusBadRequest, EngineRefused)
	}
	if engine == models.EnginePostgres && !config.Postgres.Enabled {
		return fresh, nil, shortcuts.ServiceError(http.StatusServiceUnavailable, PostgresUnavailable)
	}
	if engine == models.EngineRedis && !config.Redis.Enabled {
		return fresh, nil, shortcuts.ServiceError(http.StatusServiceUnavailable, RedisUnavailable)
	}
	if !actor.Can(permissions.ForEngine(permission, engine)) {
		return fresh, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	if !canProvision(actor, engine) {
		return fresh, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	return fresh, actor, nil
}

func canProvision(actor *access.AccountAccess, engine string) bool {
	switch engine {
	case models.EnginePostgres:
		return actor.Can(permissions.PostgresCredentialsManage) && actor.Can(permissions.PostgresDataRead) && actor.Can(permissions.PostgresDataWrite) && actor.Can(permissions.PostgresObjectsManage)
	case models.EngineRedis:
		return actor.Can(permissions.RedisCredentialsManage) && actor.Can(permissions.RedisDataRead) && actor.Can(permissions.RedisDataWrite)
	case models.EngineLibSQL:
		return true
	}
	return false
}

func selectedEngine(engine string) string {
	if engine == "" {
		return models.EngineLibSQL
	}
	return engine
}
