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
	if engine != "" && engine != models.EngineLibSQL && engine != models.EnginePostgres {
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
	return &IndexContext{Title: IndexTitle, Databases: toDatabaseViews(fresh, filtered), Engine: engine, EngineFilters: collections.OptionsOf("", "Every engine", models.EngineLibSQL, "libSQL", models.EnginePostgres, "PostgreSQL"), CreateEngines: createEngines, ImportEngines: importEngines, CanCreateDatabases: len(createEngines) > 0, CanImportDatabases: len(importEngines) > 0, CanImportLibSQL: canImportLibSQL, CanImportPostgres: canImportPostgres, CanCloneRemote: overviewhelper.CanCloneRemote(actor), RemoteTLSModes: overviewhelper.RemoteTLSModes()}, nil
}

func engineOptions(actor *access.AccountAccess, permission string) []collections.Option {
	options := make([]collections.Option, 0, 2)
	for _, engine := range []string{models.EngineLibSQL, models.EnginePostgres} {
		if engine == models.EnginePostgres && !config.Postgres.Enabled {
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
		view := DatabaseView{Name: record.Name, Engine: record.Kind(), EngineLabel: engineLabel(record.Kind()), State: stateLabel(record), Address: record.Name + "." + config.Server.Domain, Protected: record.Protected, Storage: "Unavailable", Tables: "—", RowsRead: "—", Queries: "—", Ready: record.Available(), Actions: access.Actions(actor, record)}
		fresh := engines.WithDatabase(ctx, record)
		if record.Kind() == models.EnginePostgres {
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
		if record.Available() && actor.CanDatabase(record, permissions.DataRead) {
			view.Tables = tableCount(fresh, record)
		}
		views = append(views, view)
	}
	return views
}
