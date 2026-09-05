package insights

import (
	"net/http"
	"sort"

	"sqldash/analytics"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type QueriesAsk struct {
	Database  string
	Window    string
	Sort      string
	Direction string
	Page      int
}

func GetQueries(asked QueriesAsk) (*QueriesContext, *fiber.Error) {
	window := analytics.WindowNamed(asked.Window)

	summary, summaryError := analytics.Overall(asked.Database, window)
	if summaryError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, summaryError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
	}

	every, everyError := analytics.EveryQuery(asked.Database, window)
	if everyError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, everyError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
	}

	shown := &QueriesContext{
		Title:         QueriesTitle,
		Heading:       headingFor(asked.Database),
		Scope:         scopeFor(asked.Database),
		Database:      asked.Database,
		FormAction:    queriesPathFor(asked.Database),
		QueriesPath:   queriesPathFor(asked.Database),
		Window:        window.Label,
		WindowOptions: windowOptions(),
		Sort:          sortNamed(asked.Sort),
		Direction:     directionNamed(asked.Direction),
		Total:         int64(len(every)),
		PageSize:      QueryPageSize,
		Columns:       queryColumns(),
	}

	order(every, shown.Sort, shown.Direction)

	shown.Pages = pagesFor(len(every), QueryPageSize)
	shown.Page = asked.Page

	if shown.Page < 1 {
		shown.Page = 1
	}

	if shown.Pages > 0 && shown.Page > shown.Pages {
		shown.Page = shown.Pages
	}

	from := (shown.Page - 1) * QueryPageSize
	to := from + QueryPageSize

	if from > len(every) {
		from = len(every)
	}

	if to > len(every) {
		to = len(every)
	}

	page := every[from:to]
	analytics.Name(asked.Database, page)

	shown.Rows = toQueryViews(page, summary.TotalDuration)

	if len(page) > 0 {
		shown.FirstRow = from + 1
		shown.LastRow = to
	}

	shown.PreviousPage = shown.Page - 1
	shown.NextPage = shown.Page + 1

	return shown, nil
}

func queryColumns() []QueryColumn {
	return []QueryColumn{
		{Key: ShareSort, Label: ShareHeading, Numeric: true},
		{Key: CountSort, Label: CountHeading, Numeric: true},
		{Key: TotalSort, Label: TotalHeading, Numeric: true},
		{Key: MedianSort, Label: MedianHeading, Numeric: true},
		{Key: TailSort, Label: TailHeading, Numeric: true},
		{Key: ReadSort, Label: ReadHeading, Numeric: true},
		{Key: RatioSort, Label: RatioHeading, Numeric: true},
		{Key: WrittenSort, Label: WrittenHeading, Numeric: true},
	}
}

func sortNamed(asked string) string {
	for _, column := range queryColumns() {
		if column.Key == asked {
			return asked
		}
	}

	return ShareSort
}

func directionNamed(asked string) string {
	if asked == AscendingOrder {
		return AscendingOrder
	}

	return DescendingOrder
}

func order(summaries []analytics.QuerySummary, by string, direction string) {
	sort.SliceStable(summaries, func(first int, second int) bool {
		left := weigh(summaries[first], by)
		right := weigh(summaries[second], by)

		if direction == AscendingOrder {
			return left < right
		}

		return left > right
	})
}

func weigh(summary analytics.QuerySummary, by string) float64 {
	switch by {
	case CountSort:
		return float64(summary.Count)
	case MedianSort:
		return summary.Histogram.Percentile(0.50)
	case TailSort:
		return summary.Histogram.Percentile(0.99)
	case ReadSort:
		return float64(summary.RowsRead)
	case WrittenSort:
		return float64(summary.RowsWritten)
	case RatioSort:
		if summary.RowsReturned == 0 {
			return 0
		}

		return float64(summary.RowsRead) / float64(summary.RowsReturned)
	}

	return summary.TotalDuration
}

func pagesFor(total int, size int) int {
	if total == 0 {
		return 0
	}

	pages := total / size
	if total%size != 0 {
		pages++
	}

	return pages
}

func headingFor(databaseName string) string {
	if databaseName == "" {
		return OverviewTitle
	}

	return databaseName
}

func scopeFor(databaseName string) string {
	if databaseName == "" {
		return EveryDatabase
	}

	return databaseName
}

func queriesPathFor(databaseName string) string {
	if databaseName == "" {
		return QueriesPath
	}

	return DatabasePath + databaseName + QueriesSuffix
}
