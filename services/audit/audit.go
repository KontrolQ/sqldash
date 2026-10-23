package audit

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"net/http"
	"net/url"
	"sqldash/models"
	"sqldash/permissions"
	databaseRepository "sqldash/repositories/database"
	repository "sqldash/repositories/event"
	"sqldash/services/access"
	"sqldash/utils/collections"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"
	"strconv"
	"time"
)

func Record(actor string, databaseName string, action string, detail string) {
	var record *models.Database
	if databaseName != "" {
		var findError error
		record, findError = databaseRepository.FindByName(databaseName)
		if findError == nil && record == nil {
			findError = errors.New(DatabaseIdentityUnavailable)
		}
		if findError != nil {
			logger.Errorf(LogPrefix, RecordFailedLog, action, findError)
			return
		}
	}
	RecordDatabase(actor, record, action, detail)
}

func RecordDatabase(actor string, record *models.Database, action string, detail string) {
	held := &models.Event{OccurredAt: time.Now(), Action: action, Detail: detail, Actor: actor}
	if record != nil {
		held.Database = record.Name
		held.DatabaseID = record.ID
		held.Engine = record.Kind()
	}
	if createError := repository.Create(held); createError != nil {
		logger.Errorf(LogPrefix, RecordFailedLog, action, createError)
	}
}

func PageFor(actor *access.AccountAccess, databaseName string, action string, page int, filters ...Filter) (*LogContext, *fiber.Error) {
	if page < 1 {
		page = 1
	}
	filter := Filter{}
	if len(filters) > 0 {
		filter = filters[0]
	}
	if filter.Engine != "" && filter.Engine != models.EngineLibSQL && filter.Engine != models.EnginePostgres && filter.Engine != PlatformEngine {
		return nil, shortcuts.ServiceError(http.StatusBadRequest, InvalidFilter)
	}
	if filter.DatabaseID == 0 && databaseName != "" {
		record, findError := databaseRepository.FindByName(databaseName)
		if findError != nil {
			return nil, shortcuts.ServiceError(http.StatusInternalServerError, LogUnavailable)
		}
		if record != nil {
			filter.DatabaseID = record.ID
		}
	}
	scope := repository.Scope{AllDatabases: actor.All(), Platform: actor.Can(permissions.PlatformAuditView), Engine: filter.Engine, DatabaseID: filter.DatabaseID}
	if actor.Can(permissions.AuditView) {
		scope.Engines = append(scope.Engines, models.EngineLibSQL)
	}
	if actor.Can(permissions.PostgresAuditView) {
		scope.Engines = append(scope.Engines, models.EnginePostgres)
	}
	if !scope.AllDatabases && len(scope.Engines) > 0 {
		records, findError := access.VisibleFor(actor, permissions.AuditView)
		if findError != nil {
			return nil, shortcuts.ServiceError(http.StatusInternalServerError, LogUnavailable)
		}
		for _, record := range records {
			scope.DatabaseIDs = append(scope.DatabaseIDs, record.ID)
		}
	}
	entries, total, findError := repository.PageScoped(scope, databaseName, action, (page-1)*PageSize, PageSize)
	if findError != nil {
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, LogUnavailable)
	}
	actions, actionsError := repository.ActionsScoped(scope, databaseName)
	if actionsError != nil {
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, LogUnavailable)
	}
	shown := logContext(databaseName, action, page, entries, total, actions)
	shown.Engine = filter.Engine
	shown.DatabaseID = filter.DatabaseID
	shown.EngineChoices = engineChoicesFor(scope)
	shown.Carry = carryOf(databaseName, action, filter)
	if filter.DatabaseID != 0 && len(entries) > 0 {
		shown.Database = entries[0].Database
	}
	return shown, nil
}

func Page(databaseName string, action string, page int, allowed ...[]string) (*LogContext, *fiber.Error) {
	if page < 1 {
		page = 1
	}
	records, total, findError := repository.Page(databaseName, action, (page-1)*PageSize, PageSize, allowed...)
	if findError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, LogUnavailable)
	}
	actions, actionsError := repository.Actions(databaseName, allowed...)
	if actionsError != nil {
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, LogUnavailable)
	}
	return logContext(databaseName, action, page, records, total, actions), nil
}

func logContext(databaseName string, action string, page int, records []models.Event, total int64, actions []string) *LogContext {
	shown := &LogContext{Title: Title, Heading: Title, Database: databaseName, Action: action, Page: page, Total: total, Pages: int((total + PageSize - 1) / PageSize), Choices: choicesFor(actions), Carry: carryOf(databaseName, action)}
	if shown.Pages > 0 && shown.Page > shown.Pages {
		shown.Page = shown.Pages
	}
	shown.PreviousPage = shown.Page - 1
	shown.NextPage = shown.Page + 1
	for _, record := range records {
		shown.Entries = append(shown.Entries, EntryView{When: record.OccurredAt.Local().Format(MomentLayout), Ago: readableAgo(record.OccurredAt), Database: record.Database, DatabaseID: record.DatabaseID, DatabaseLink: carryOf(record.Database, "", Filter{DatabaseID: record.DatabaseID, Engine: record.Engine}), Engine: engineLabel(record.Engine), Action: record.Action, Label: LabelFor(record.Action), Tone: toneOf(record.Action), Detail: record.Detail, Actor: record.Actor})
	}
	if total > 0 {
		shown.FirstRow = (shown.Page-1)*PageSize + 1
		shown.LastRow = shown.FirstRow + len(shown.Entries) - 1
	}
	return shown
}

func choicesFor(actions []string) []collections.Option {
	choices := []collections.Option{{Value: "", Label: EveryAction}}
	for _, action := range actions {
		choices = append(choices, collections.Option{Value: action, Label: LabelFor(action)})
	}
	return choices
}

func carryOf(databaseName string, action string, filters ...Filter) string {
	values := url.Values{}
	filter := Filter{}
	if len(filters) > 0 {
		filter = filters[0]
	}
	if filter.DatabaseID != 0 {
		values.Set(DatabaseIDParameter, strconv.FormatUint(uint64(filter.DatabaseID), 10))
	} else if databaseName != "" {
		values.Set(DatabaseParameter, databaseName)
	}
	if filter.Engine != "" {
		values.Set(EngineParameter, filter.Engine)
	}
	if action != "" {
		values.Set(ActionParameter, action)
	}
	if len(values) == 0 {
		return ""
	}
	return values.Encode() + "&"
}

func engineChoicesFor(scope repository.Scope) []collections.Option {
	choices := []collections.Option{{Value: "", Label: EveryEngine}}
	for _, engine := range scope.Engines {
		choices = append(choices, collections.Option{Value: engine, Label: engineLabel(engine)})
	}
	if scope.Platform {
		choices = append(choices, collections.Option{Value: PlatformEngine, Label: engineLabel(PlatformEngine)})
	}
	return choices
}

func engineLabel(engine string) string {
	switch engine {
	case models.EngineLibSQL:
		return "libSQL"
	case models.EnginePostgres:
		return "PostgreSQL"
	case "", PlatformEngine:
		return "Administration"
	default:
		return engine
	}
}

func readableAgo(when time.Time) string {
	gap := time.Since(when)
	if gap < time.Minute {
		return JustNow
	}
	if gap < time.Hour {
		return plural(int(gap.Minutes()), MinuteWord)
	}
	if gap < DayHours*time.Hour {
		return plural(int(gap.Hours()), HourWord)
	}
	return plural(int(gap.Hours()/DayHours), DayWord)
}
