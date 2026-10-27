package databases

import (
	"context"
	"net/http"
	"strconv"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/redishost"
	"sqldash/services/access"
	"sqldash/services/databases/overviewhelper"
	"sqldash/utils/collections"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetIndexData(ctx context.Context) (*IndexContext, *fiber.Error) {
	return GetIndexDataForEngine(ctx, "")
}

func GetIndexDataForEngine(ctx context.Context, engine string) (*IndexContext, *fiber.Error) {
	fresh, actor, err := FreshActor(ctx)
	if err != nil {
		return nil, err
	}
	if engine != "" && !models.IsKnownEngine(engine) {
		return nil, shortcuts.ServiceError(http.StatusBadRequest, EngineRefused)
	}
	records, findError := access.VisibleFor(actor, permissions.DatabaseView)
	if findError != nil {
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}
	filtered := make([]models.Database, 0, len(records))
	for _, record := range records {
		if engine == "" || record.Kind() == engine {
			filtered = append(filtered, record)
		}
	}
	createEngines := engineOptions(actor, permissions.DatabaseCreate)
	importEngines := engineOptions(actor, permissions.DatabaseImport)
	canImportLibSQL := actor.Can(permissions.DatabaseCreate) && actor.Can(permissions.DatabaseImport)
	canImportPostgres := config.Postgres.Enabled && actor.Can(permissions.PostgresDatabaseCreate) && actor.Can(permissions.PostgresDatabaseImport)
	canImportRedis := config.Redis.Enabled && actor.Can(permissions.RedisDatabaseCreate) && actor.Can(permissions.RedisDatabaseImport) && canProvision(actor, models.EngineRedis)
	objectHeading := "Tables"
	if engine == models.EngineRedis {
		objectHeading = "Keys"
	} else if engine == "" {
		objectHeading = "Tables / keys"
	}
	return &IndexContext{
		Title: IndexTitle, Databases: toDatabaseViews(fresh, filtered), Engine: engine,
		EngineFilters: collections.OptionsOf("", "Every engine", models.EngineLibSQL, "libSQL", models.EnginePostgres, "PostgreSQL", models.EngineRedis, "Redis"),
		CreateEngines: createEngines, ImportEngines: importEngines, CanCreateDatabases: len(createEngines) > 0, CanImportDatabases: len(importEngines) > 0,
		CanImportLibSQL: canImportLibSQL, CanImportPostgres: canImportPostgres, CanImportRedis: canImportRedis, ObjectHeading: objectHeading,
		CanCloneRemote: overviewhelper.CanCloneRemote(actor), RemoteTLSModes: overviewhelper.RemoteTLSModes(),
	}, nil
}

func engineOptions(actor *access.AccountAccess, permission string) []collections.Option {
	options := make([]collections.Option, 0, 3)
	for _, engine := range []string{models.EngineLibSQL, models.EnginePostgres, models.EngineRedis} {
		if engine == models.EnginePostgres && !config.Postgres.Enabled || engine == models.EngineRedis && !config.Redis.Enabled {
			continue
		}
		if !canProvision(actor, engine) {
			continue
		}
		if !actor.Can(permissions.ForEngine(permission, engine)) || !actor.Can(permissions.ForEngine(permissions.DatabaseCreate, engine)) {
			continue
		}
		options = append(options, collections.OptionsOf(engine, engineLabel(engine))...)
	}
	return options
}

func engineLabel(engine string) string {
	if engine == models.EngineRedis {
		return "Redis"
	}
	if engine == models.EnginePostgres {
		return "PostgreSQL"
	}
	if engine == models.EngineLibSQL {
		return "libSQL"
	}
	return "Unavailable engine"
}

func stateLabel(record *models.Database) string {
	switch record.State {
	case "", models.DatabaseReady:
		return "Ready"
	case models.DatabasePending:
		return "Pending"
	case models.DatabaseApplying:
		return "In progress"
	case models.DatabaseDeleting:
		return "Deleting"
	default:
		return "Needs recovery"
	}
}

func toDatabaseViews(ctx context.Context, records []models.Database) []DatabaseView {
	views := make([]DatabaseView, 0, len(records))
	actor := access.FromContext(ctx)
	for index := range records {
		record := &records[index]
		view := DatabaseView{Name: record.Name, DisplayName: record.Label(), Engine: record.Kind(), EngineLabel: engineLabel(record.Kind()), State: stateLabel(record), Address: record.Name + "." + config.Server.Domain, Protected: record.Protected, Storage: "Unavailable", Tables: "—", Objects: "—", RowsRead: "—", Queries: "—", Ready: record.Available(), Actions: access.Actions(actor, record)}
		fresh := engines.WithDatabase(ctx, record)
		if record.Kind() == models.EngineRedis {
			view.Address = config.RedisHost() + ":" + strconv.Itoa(config.Redis.PublicPort)
			if record.Available() {
				if statistics, err := engines.Statistics(fresh, record.Name); err == nil {
					view.Storage = readableSize(statistics.StorageBytesUsed)
					view.Online = true
				}
				if actor.CanDatabase(record, permissions.DataRead) {
					if client, err := redishost.Client(fresh, record, redishost.ReadOnly); err == nil {
						if count, err := client.DBSize(fresh).Result(); err == nil {
							view.Objects = readableCount(count)
						}
						client.Close()
					}
				}
			}
		} else if record.Kind() == models.EnginePostgres {
			view.Address = config.PostgresHost() + ":" + strconv.Itoa(config.Postgres.PublicPort)
			if record.Available() {
				if statistics, err := control.Statistics(fresh, record); err == nil {
					view.Storage = readableSize(statistics.StorageBytes)
					view.Online = true
				}
			}
		} else if record.Available() {
			if statistics, err := engines.Statistics(fresh, record.Name); err == nil {
				view.Storage = readableSize(statistics.StorageBytesUsed)
				view.RowsRead = readableCount(statistics.RowsRead)
				view.Queries = readableCount(statistics.QueryCount)
				view.Online = true
			}
			if settings, err := engines.Configuration(fresh, record.Name); err == nil {
				view.Blocked = settings.BlockReads || settings.BlockWrites
			}
		}
		if record.Kind() != models.EngineRedis && record.Available() && actor.CanDatabase(record, permissions.DataRead) {
			view.Tables = tableCount(fresh, record)
			view.Objects = view.Tables
		}
		views = append(views, view)
	}
	return views
}
