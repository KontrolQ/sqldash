package insights

import (
	"context"
	"net/http"
	"time"

	"sqldash/analytics"
	"sqldash/config"
	"sqldash/models"
	"sqldash/pgmetrics"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func scopedQueryTable(ctx context.Context, asked QueriesAsk, engine string) (*QueriesContext, *fiber.Error) {
	fresh, records, err := insightRecords(ctx, asked.Database)
	if err != nil {
		return nil, err
	}
	options := insightEngineOptions(access.FromContext(fresh), config.Postgres.Enabled)
	engine, err = selectedInsightEngine(engine, records, asked.Database, options)
	if err != nil {
		return nil, err
	}
	window := analytics.WindowNamed(asked.Window)
	until := time.Now().UTC()
	since := until.Add(-window.Since)
	libsqlNames, nativeRecords := splitInsightRecords(records, engine)
	shown := &QueriesContext{Title: QueriesTitle, Heading: headingFor(asked.Database), Scope: scopeFor(asked.Database), Database: asked.Database, FormAction: queriesPathFor(asked.Database), QueriesPath: queriesPathFor(asked.Database), Window: window.Label, WindowOptions: windowOptions(), Engine: engine, EngineOptions: options, Global: asked.Database == "", Sort: queryTableSortNamed(asked.Sort), Direction: directionNamed(asked.Direction)}
	rows := make([]QueryTableRow, 0)
	if includesInsightEngine(engine, models.EngineLibSQL, options) {
		shown.ShowLibSQL = true
		shown.HasDatabases = len(libsqlNames) > 0
		summaries, readError := analytics.DatabaseQueries(fresh, asked.Database, window.Resolution, since, until, libsqlNames)
		if readError != nil {
			return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
		}
		rows = append(rows, libSQLTableRows(summaries)...)
	}
	if includesInsightEngine(engine, models.EnginePostgres, options) {
		shown.HasDatabases = shown.HasDatabases || len(nativeRecords) > 0
		history, readError := pgmetrics.ReadHistory(fresh, nativeRecords, since, until)
		if readError != nil {
			return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
		}
		shown.Native = nativeQueryState(nativeRecords, history, asked, until)
		rows = append(rows, nativeTableRows(history.Queries)...)
	}
	if includesInsightEngine(engine, models.EngineRedis, options) {
		shown.Redis = redisInsights(fresh, records, true)
	}
	orderQueryTable(rows, shown.Sort, shown.Direction)
	paginateQueryTable(shown, rows, asked.Page)
	return shown, nil
}

func nativeQueryState(records []models.Database, history *pgmetrics.History, asked QueriesAsk, until time.Time) *NativeContext {
	shown := nativeContextFor(records, asked)
	shown.HasSamples = len(history.Latest) > 0
	shown.HasIntervals = history.Intervals > 0
	shown.Intervals, shown.Discontinuities = history.Intervals, history.Discontinuities
	for _, sample := range history.Latest {
		shown.HasStatementSource = shown.HasStatementSource || sample.StatementsAvailable
		shown.HasFreshSamples = shown.HasFreshSamples || until.Sub(sample.CollectedAt) <= 3*pgmetrics.SampleInterval
	}
	if history.FirstCollectedAt != nil {
		shown.FirstCollectedAt = history.FirstCollectedAt.Local().Format(time.DateTime)
	}
	if history.LastCollectedAt != nil {
		shown.CollectedAt = history.LastCollectedAt.Local().Format(time.DateTime)
	}
	setNativeEmptyState(shown)
	return shown
}
