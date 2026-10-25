package redis

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"sqldash/database"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redisdata"
	"sqldash/redishost"
	"sqldash/services/access"
	"sqldash/services/databases"

	"github.com/gofiber/fiber/v3"
	redisclient "github.com/redis/go-redis/v9"
)

func authorize(ctx context.Context, name, permission string, ready bool) (context.Context, *access.AccountAccess, *models.Database, *fiber.Error) {
	fresh, actor, err := databases.FreshActor(ctx)
	if err != nil {
		return fresh, nil, nil, fiber.NewError(403, AuthorityRefused)
	}
	record, findError := access.Database(actor, name)
	if findError != nil {
		return fresh, nil, nil, fiber.NewError(404, DatabaseMissing)
	}
	if record.Kind() != models.EngineRedis {
		return fresh, nil, nil, fiber.NewError(400, "This action requires a Redis database.")
	}
	if !actor.CanDatabase(record, permission) {
		return fresh, nil, nil, fiber.NewError(403, AuthorityRefused)
	}
	if ready && !record.Available() {
		return fresh, nil, nil, fiber.NewError(409, DatabaseNotReady)
	}
	return engines.WithDatabase(fresh, record), actor, record, nil
}

func dataClient(ctx context.Context, name string, write bool) (context.Context, *redisclient.Client, *fiber.Error) {
	permission, profile := permissions.RedisDataRead, redishost.ReadOnly
	if write {
		permission, profile = permissions.RedisDataWrite, redishost.ReadWrite
	}
	fresh, actor, record, err := authorize(ctx, name, permission, true)
	if err != nil {
		return fresh, nil, err
	}
	if !actor.CanDatabase(record, permissions.RedisDataRead) {
		return fresh, nil, fiber.NewError(403, AuthorityRefused)
	}
	var instance models.RedisInstance
	if database.DB.WithContext(fresh).Where("database_id = ? AND identity = ?", record.ID, record.ClusterID).First(&instance).Error != nil || instance.State != models.DatabaseReady {
		return fresh, nil, fiber.NewError(409, DatabaseNotReady)
	}
	key, keyError := redishost.ReferenceKey(fresh, record)
	if keyError != nil || len(key) < 32 {
		return fresh, nil, fiber.NewError(503, RedisUnavailable)
	}
	defer clear(key)
	fresh = redisdata.WithScope(fresh, redisdata.Scope{DatabaseID: strconv.FormatUint(uint64(record.ID), 10), Generation: instance.Identity + ":" + strconv.FormatUint(instance.Generation, 10), SigningKey: key})
	client, clientError := redishost.Client(fresh, record, profile)
	if clientError != nil {
		return fresh, nil, fiber.NewError(503, RedisUnavailable)
	}
	var current models.RedisInstance
	if database.DB.WithContext(fresh).Where("database_id = ? AND identity = ? AND generation = ? AND state = ?", record.ID, instance.Identity, instance.Generation, models.DatabaseReady).First(&current).Error != nil {
		client.Close()
		return fresh, nil, fiber.NewError(409, "The Redis instance changed while opening this request. Inspect the key again.")
	}
	return fresh, client, nil
}

func operationError(err error) *fiber.Error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, redisdata.InvalidVersion):
		return fiber.NewError(409, "The Redis key reference is stale or no longer valid. Inspect the current value before retrying.")
	case errors.Is(err, redisdata.InvalidInput), errors.Is(err, redisdata.Unsupported):
		return fiber.NewError(400, err.Error())
	case errors.Is(err, redisdata.StaleValue), errors.Is(err, redisdata.TargetExists):
		return fiber.NewError(409, err.Error())
	case errors.Is(err, redisdata.LimitExceeded):
		return fiber.NewError(413, err.Error())
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return fiber.NewError(504, "Redis request timed out. Inspect the current value before retrying a write.")
	case strings.HasPrefix(err.Error(), "NOPERM"), strings.HasPrefix(err.Error(), "READONLY"):
		return fiber.NewError(403, ReadOnly)
	default:
		return fiber.NewError(503, RedisUnavailable)
	}
}
