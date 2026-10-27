package insights

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"sqldash/analytics"
	"sqldash/config"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgmetrics"
	"sqldash/services/access"
	"sqldash/services/databases"
	"sqldash/utils/collections"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type NativeQueryView struct {
	Key              string
	Database         string
	Statement        string
	QueryID          string
	Calls            string
	Rows             string
	TotalTime        string
	MeanTime         string
	MinimumTime      string
	MaximumTime      string
	SharedHit        string
	SharedRead       string
	LocalHit         string
	LocalRead        string
	TemporaryRead    string
	TemporaryWritten string
	Share            string
}

type NativeDatabaseView struct {
	Name                string
	Path                string
	Storage             string
	Connections         int64
	CollectedAt         string
	StatisticsReset     string
	StatementReset      string
	PostmasterStarted   string
	StatementsAvailable bool
	Stale               bool
}

type NativeContext struct {
	Rows               []NativeQueryView
	Databases          []NativeDatabaseView
	Tiles              []TileView
	CallsChart         string
	ExecutionChart     string
	StorageChart       string
	ConnectionsChart   string
	ChartCoverage      string
	Calls              string
	RowsProcessed      string
	ExecutionTime      string
	MeanExecutionTime  string
	SharedHit          string
	SharedRead         string
	Committed          string
	RolledBack         string
	Deadlocks          string
	Discontinuities    int64
	Intervals          int64
	CollectedAt        string
	FirstCollectedAt   string
	HasDatabases       bool
	HasReadyDatabases  bool
	HasSamples         bool
	HasFreshSamples    bool
	HasIntervals       bool
	HasStatementSource bool
	EmptyTitle         string
	EmptyNote          string
	Total              int
	Page               int
	Pages              int
	FirstRow           int
	LastRow            int
	PreviousPage       int
	NextPage           int
	Sort               string
	Direction          string
	Columns            []QueryColumn
}

func GetScopedOverview(ctx context.Context, name string, windowLabel string, engine string) (*OverviewContext, *fiber.Error) {
	fresh, records, err := insightRecords(ctx, name)
	if err != nil {
		return nil, err
	}
	options := insightEngineOptions(access.FromContext(fresh), config.Postgres.Enabled)
	engine, err = selectedInsightEngine(engine, records, name, options)
	if err != nil {
		return nil, err
	}
	libsqlNames, nativeRecords := splitInsightRecords(records, engine)
	shown := &OverviewContext{Title: headingFor(name), Heading: headingFor(name), Scope: scopeFor(name), FormAction: overviewPathFor(name), QueriesPath: queriesPathFor(name), Window: analytics.WindowNamed(windowLabel).Label, Windows: windowLabels(), WindowOptions: windowOptions()}
	if includesInsightEngine(engine, models.EngineLibSQL, options) {
		shown, err = GetOverview(name, windowLabel, libsqlNames)
		if err != nil {
			return nil, err
		}
		shown.ShowLibSQL = true
	}
	shown.Engine = engine
	shown.Global = name == ""
	shown.EngineOptions = options
	if includesInsightEngine(engine, models.EnginePostgres, options) {
		shown.Native, err = nativeInsights(fresh, nativeRecords, windowLabel, QueriesAsk{Sort: TotalSort, Direction: DescendingOrder, Page: 1}, TopQueryLimit, true)
		if err != nil {
			return nil, err
		}
	}
	if includesInsightEngine(engine, models.EngineRedis, options) {
		shown.Redis = redisInsights(fresh, records, false)
	}
	return shown, nil
}

func GetScopedQueries(ctx context.Context, asked QueriesAsk, engine string) (*QueriesContext, *fiber.Error) {
	return scopedQueryTable(ctx, asked, engine)
}

func insightRecords(ctx context.Context, name string) (context.Context, []models.Database, *fiber.Error) {
	if name != "" {
		fresh, record, err := databases.Authorize(ctx, name, permissions.InsightsView, false)
		if err != nil {
			return fresh, nil, err
		}
		return fresh, []models.Database{*record}, nil
	}
	fresh, actor, err := databases.FreshActor(ctx)
	if err != nil {
		return fresh, nil, err
	}
	records, findError := access.VisibleFor(actor, permissions.InsightsView)
	if findError != nil {
		return fresh, nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
	}
	return fresh, records, nil
}

func selectedInsightEngine(engine string, records []models.Database, name string, options []collections.Option) (string, *fiber.Error) {
	if name != "" && len(records) == 1 {
		engine = records[0].Kind()
	}
	if !models.IsKnownEngine(engine) {
		return AllEngines, nil
	}
	if includesInsightEngine(engine, engine, options) {
		return engine, nil
	}
	if engine == models.EngineRedis && !config.Redis.Enabled {
		return "", shortcuts.ServiceError(http.StatusServiceUnavailable, "Redis is unavailable.")
	}
	if engine == models.EnginePostgres && !config.Postgres.Enabled {
		return "", shortcuts.ServiceError(http.StatusServiceUnavailable, PostgresUnavailable)
	}
	return "", shortcuts.ServiceError(http.StatusForbidden, EngineAuthorityRefused)
}

func includesInsightEngine(selected string, engine string, options []collections.Option) bool {
	if selected != AllEngines && selected != engine {
		return false
	}
	for _, option := range options {
		if option.Value == engine {
			return true
		}
	}
	return false
}

func splitInsightRecords(records []models.Database, engine string) ([]string, []models.Database) {
	libsqlNames := []string{}
	nativeRecords := []models.Database{}
	for _, record := range records {
		if engine != AllEngines && engine != record.Kind() {
			continue
		}
		if record.Kind() == models.EngineLibSQL && record.Available() {
			libsqlNames = append(libsqlNames, record.Name)
		}
		if record.Kind() == models.EnginePostgres {
			nativeRecords = append(nativeRecords, record)
		}
	}
	return libsqlNames, nativeRecords
}

func insightEngineOptions(actor *access.AccountAccess, postgresEnabled bool) []collections.Option {
	choices := []collections.Option{{Value: AllEngines, Label: AllEnginesLabel}}
	if actor.Can(permissions.InsightsView) {
		choices = append(choices, collections.Option{Value: models.EngineLibSQL, Label: LibSQLLabel})
	}
	if postgresEnabled && actor.Can(permissions.PostgresInsightsView) {
		choices = append(choices, collections.Option{Value: models.EnginePostgres, Label: PostgresLabel})
	}
	if config.Redis.Enabled && actor.Can(permissions.RedisInsightsView) {
		choices = append(choices, collections.Option{Value: models.EngineRedis, Label: "Redis"})
	}
	return choices
}

func overviewPathFor(name string) string {
	if name == "" {
		return OverviewPath
	}
	return DatabasePath + url.PathEscape(name) + InsightsSuffix
}

func nativeInsights(ctx context.Context, records []models.Database, windowLabel string, asked QueriesAsk, limit int, withCharts bool) (*NativeContext, *fiber.Error) {
	until := time.Now().UTC()
	window := analytics.WindowNamed(windowLabel)
	since := until.Add(-window.Since)
	shown := nativeContextFor(records, asked)
	if !shown.HasReadyDatabases {
		if withCharts {
			applyNativeCharts(shown, pgmetrics.ReduceSeries(records, nil, nil, since, until, analytics.SeriesSlots))
		}
		setNativeEmptyState(shown)
		return shown, nil
	}
	held, err := pgmetrics.ReadHistory(ctx, records, since, until)
	if err != nil {
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
	}
	if withCharts {
		series, seriesError := pgmetrics.ReadSeries(ctx, records, since, until, analytics.SeriesSlots)
		if seriesError != nil {
			return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
		}
		applyNativeCharts(shown, series)
	}
	shown.Discontinuities = held.Discontinuities
	shown.Intervals = held.Intervals
	shown.HasSamples = len(held.Latest) > 0
	shown.HasIntervals = held.Intervals > 0
	shown.Total = len(held.Queries)
	if shown.HasIntervals {
		shown.Calls = readableCount(held.Calls)
		shown.RowsProcessed = readableCount(held.Rows)
		shown.ExecutionTime = readableDuration(held.ExecutionMs)
		shown.SharedHit = readableCount(held.SharedBlocksHit)
		shown.SharedRead = readableCount(held.SharedBlocksRead)
		if held.Calls > 0 {
			shown.MeanExecutionTime = readableDuration(held.ExecutionMs / float64(held.Calls))
		}
	}
	if hasNativeDatabaseIntervals(held, since, until) {
		shown.Committed = readableCount(held.Committed)
		shown.RolledBack = readableCount(held.RolledBack)
		shown.Deadlocks = readableCount(held.Deadlocks)
	}
	if held.LastCollectedAt != nil {
		shown.CollectedAt = held.LastCollectedAt.Local().Format(time.DateTime)
	}
	if held.FirstCollectedAt != nil {
		shown.FirstCollectedAt = held.FirstCollectedAt.Local().Format(time.DateTime)
	}
	byID := make(map[uint]models.Database, len(records))
	for _, record := range records {
		byID[record.ID] = record
	}
	for _, sample := range held.Latest {
		shown.HasStatementSource = shown.HasStatementSource || sample.StatementsAvailable
		stale := until.Sub(sample.CollectedAt) > 3*pgmetrics.SampleInterval
		shown.HasFreshSamples = shown.HasFreshSamples || !stale
		record := byID[sample.DatabaseID]
		shown.Databases = append(shown.Databases, NativeDatabaseView{Name: record.Label(), Path: overviewPathFor(record.Name), Storage: readableBytes(sample.StorageBytes), Connections: sample.Connections,
			CollectedAt: sample.CollectedAt.Local().Format(time.DateTime), StatisticsReset: optionalMoment(sample.ResetAt), StatementReset: optionalMoment(sample.StatementResetAt),
			PostmasterStarted: sample.PostmasterStartedAt.Local().Format(time.DateTime), StatementsAvailable: sample.StatementsAvailable, Stale: stale})
	}
	sort.SliceStable(held.Queries, func(first int, second int) bool {
		left, right := nativeWeight(held.Queries[first], shown.Sort), nativeWeight(held.Queries[second], shown.Sort)
		if left == right {
			return held.Queries[first].Key() < held.Queries[second].Key()
		}
		if shown.Direction == AscendingOrder {
			return left < right
		}
		return left > right
	})
	shown.Pages = pagesFor(len(held.Queries), limit)
	shown.Page = asked.Page
	if shown.Page < 1 {
		shown.Page = 1
	}
	if shown.Pages == 0 {
		shown.Page = 1
	} else if shown.Page > shown.Pages {
		shown.Page = shown.Pages
	}
	from := (shown.Page - 1) * limit
	if from > len(held.Queries) {
		from = len(held.Queries)
	}
	to := min(from+limit, len(held.Queries))
	for _, query := range held.Queries[from:to] {
		share := 0.0
		if held.ExecutionMs > 0 {
			share = query.ExecutionMs / held.ExecutionMs * 100
		}
		shown.Rows = append(shown.Rows, NativeQueryView{Key: query.Key(), Database: query.DatabaseName, Statement: query.Statement, QueryID: query.QueryID,
			Calls: readableCount(query.Calls), Rows: readableCount(query.Rows), TotalTime: readableDuration(query.ExecutionMs), MeanTime: readableDuration(query.MeanExecutionMs()),
			MinimumTime: readableDuration(query.MinimumExecutionMs), MaximumTime: readableDuration(query.MaximumExecutionMs), SharedHit: readableCount(query.SharedBlocksHit),
			SharedRead: readableCount(query.SharedBlocksRead), LocalHit: readableCount(query.LocalBlocksHit), LocalRead: readableCount(query.LocalBlocksRead),
			TemporaryRead: readableCount(query.TemporaryBlocksRead), TemporaryWritten: readableCount(query.TemporaryBlocksWritten), Share: fmt.Sprintf(ShareFormat, share)})
	}
	if len(shown.Rows) > 0 {
		shown.FirstRow = from + 1
		shown.LastRow = to
	}
	shown.PreviousPage, shown.NextPage = shown.Page-1, shown.Page+1
	setNativeEmptyState(shown)
	return shown, nil
}

func nativeContextFor(records []models.Database, asked QueriesAsk) *NativeContext {
	shown := &NativeContext{Calls: UnavailableValue, RowsProcessed: UnavailableValue, ExecutionTime: UnavailableValue, MeanExecutionTime: UnavailableValue,
		SharedHit: UnavailableValue, SharedRead: UnavailableValue, Committed: UnavailableValue,
		RolledBack: UnavailableValue, Deadlocks: UnavailableValue,
		HasDatabases: len(records) > 0, Page: 1, Sort: nativeSortNamed(asked.Sort), Direction: directionNamed(asked.Direction), Columns: nativeQueryColumns()}
	for _, record := range records {
		shown.HasReadyDatabases = shown.HasReadyDatabases || record.Available()
	}
	return shown
}

func hasNativeDatabaseIntervals(held *pgmetrics.History, since time.Time, until time.Time) bool {
	if held.Intervals > 0 || held.Committed > 0 || held.RolledBack > 0 || held.Deadlocks > 0 {
		return true
	}
	for _, sample := range held.Latest {
		if sample.Continuous && sample.IntervalStartedAt != nil && !sample.IntervalStartedAt.Before(since) && !sample.CollectedAt.Before(since) && !sample.CollectedAt.After(until) {
			return true
		}
	}
	return false
}

func setNativeEmptyState(shown *NativeContext) {
	switch {
	case !shown.HasDatabases:
		shown.EmptyTitle, shown.EmptyNote = NoPostgresDatabasesTitle, NoPostgresDatabasesNote
	case !shown.HasReadyDatabases:
		shown.EmptyTitle, shown.EmptyNote = PostgresNotReadyTitle, PostgresNotReadyNote
	case !shown.HasSamples:
		shown.EmptyTitle, shown.EmptyNote = PostgresWaitingTitle, PostgresWaitingNote
	case !shown.HasFreshSamples && !shown.HasIntervals:
		shown.EmptyTitle, shown.EmptyNote = PostgresStaleTitle, PostgresStaleNote
	case !shown.HasStatementSource && !shown.HasIntervals:
		shown.EmptyTitle, shown.EmptyNote = StatementsDisabledTitle, StatementsDisabledNote
	case !shown.HasIntervals:
		shown.EmptyTitle, shown.EmptyNote = StatementsWaitingTitle, StatementsWaitingNote
	default:
		shown.EmptyTitle, shown.EmptyNote = NoPostgresQueriesTitle, NoPostgresQueriesNote
	}
}

func optionalMoment(moment *time.Time) string {
	if moment == nil {
		return "—"
	}
	return moment.Local().Format(time.DateTime)
}

func nativeQueryColumns() []QueryColumn {
	return []QueryColumn{{Key: ShareSort, Label: "% of execution", Numeric: true}, {Key: CountSort, Label: "Calls", Numeric: true}, {Key: TotalSort, Label: "Execution time", Numeric: true},
		{Key: "mean", Label: "Mean execution", Numeric: true}, {Key: "rows", Label: "Rows processed", Numeric: true}, {Key: "hit", Label: "Shared blocks hit", Numeric: true}, {Key: "blocks", Label: "Shared blocks read", Numeric: true}}
}

func nativeSortNamed(asked string) string {
	for _, column := range nativeQueryColumns() {
		if column.Key == asked {
			return asked
		}
	}
	return TotalSort
}

func nativeWeight(query pgmetrics.QuerySummary, by string) float64 {
	switch by {
	case CountSort:
		return float64(query.Calls)
	case "mean":
		return query.MeanExecutionMs()
	case "rows":
		return float64(query.Rows)
	case "hit":
		return float64(query.SharedBlocksHit)
	case "blocks":
		return float64(query.SharedBlocksRead)
	default:
		return query.ExecutionMs
	}
}
