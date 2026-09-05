package insights

import (
	"fmt"
	"net/http"

	"sqldash/analytics"
	"sqldash/utils/collections"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetOverview(databaseName string, windowLabel string) (*OverviewContext, *fiber.Error) {
	window := analytics.WindowNamed(windowLabel)

	summary, summaryError := analytics.Overall(databaseName, window)
	if summaryError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, summaryError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
	}

	top, topError := analytics.TopQueries(databaseName, window, TopQueryLimit)
	if topError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, topError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, MeasurementsUnavailable)
	}

	scope := databaseName
	heading := databaseName
	action := DatabasePath + databaseName + InsightsSuffix

	if databaseName == "" {
		scope = EveryDatabase
		heading = OverviewTitle
		action = OverviewPath
	}

	shown := &OverviewContext{
		Title:         heading,
		Heading:       heading,
		FormAction:    action,
		Scope:         scope,
		Window:        window.Label,
		Windows:       windowLabels(),
		WindowOptions: windowOptions(),
		Queries:       summary.Count,
		Failures:      summary.Failures,
		RowsRead:      summary.RowsRead,
		RowsWritten:   summary.RowsWritten,
		Average:       readableDuration(summary.Average()),
		P50:           readableDuration(summary.Histogram.Percentile(0.50)),
		P75:           readableDuration(summary.Histogram.Percentile(0.75)),
		P90:           readableDuration(summary.Histogram.Percentile(0.90)),
		P95:           readableDuration(summary.Histogram.Percentile(0.95)),
		P99:           readableDuration(summary.Histogram.Percentile(0.99)),
		Approximate:   summary.Count > 0,
	}

	shown.Top = toQueryViews(top, summary.TotalDuration)

	return shown, nil
}

func toQueryViews(summaries []analytics.QuerySummary, allTime float64) []QueryView {
	views := make([]QueryView, 0, len(summaries))

	for _, summary := range summaries {
		share := 0.0
		if allTime > 0 {
			share = summary.TotalDuration / allTime * 100
		}

		views = append(views, QueryView{
			Statement:   statementOf(summary),
			Count:       summary.Count,
			TotalTime:   readableDuration(summary.TotalDuration),
			Share:       fmt.Sprintf(ShareFormat, share),
			P50:         readableDuration(summary.Histogram.Percentile(0.50)),
			P99:         readableDuration(summary.Histogram.Percentile(0.99)),
			RowsRead:    summary.RowsRead,
			RowsWritten: summary.RowsWritten,
			ReadPerRow:  ratioOf(summary.RowsRead, summary.RowsReturned),
		})
	}

	return views
}

func statementOf(summary analytics.QuerySummary) string {
	if summary.Normalised != "" {
		return summary.Normalised
	}

	return summary.Digest
}

func ratioOf(read int64, returned int64) string {
	if returned == 0 {
		return NoRatio
	}

	return fmt.Sprintf(RatioFormat, float64(read)/float64(returned))
}

func readableDuration(milliseconds float64) string {
	if milliseconds >= MillisecondsInSecond {
		return fmt.Sprintf(SecondFormat, milliseconds/MillisecondsInSecond)
	}

	return fmt.Sprintf(MillisecondFormat, milliseconds)
}

func windowLabels() []string {
	labels := make([]string, 0)

	for _, window := range analytics.Windows() {
		labels = append(labels, window.Label)
	}

	return labels
}

func windowOptions() []collections.Option {
	options := make([]collections.Option, 0)

	for _, window := range analytics.Windows() {
		options = append(options, collections.Option{Value: window.Label, Label: LastPrefix + window.Label})
	}

	return options
}
