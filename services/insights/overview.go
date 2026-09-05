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

	action := DatabasePath + databaseName + InsightsSuffix

	if databaseName == "" {
		action = OverviewPath
	}

	shown := &OverviewContext{
		Title:         headingFor(databaseName),
		Heading:       headingFor(databaseName),
		FormAction:    action,
		QueriesPath:   queriesPathFor(databaseName),
		Scope:         scopeFor(databaseName),
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

	before, beforeError := analytics.Preceding(databaseName, window)
	if beforeError != nil {
		before = &analytics.Summary{}
	}

	points, seriesError := analytics.SeriesFor(databaseName, window)
	if seriesError != nil {
		points = nil
	}

	shown.Tiles = tilesFor(summary, before, points)

	if points != nil {
		shown.TrafficChart = trafficChart(points)
		shown.LatencyChart = latencyChart(points)
		shown.TrafficSeries = []string{ReadsName, WritesName}
		shown.LatencySeries = []string{P50Name, P95Name, P99Name}
	}

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
			Example:     summary.Example,
			Count:       readableCount(summary.Count),
			TotalTime:   readableDuration(summary.TotalDuration),
			Share:       fmt.Sprintf(ShareFormat, share),
			P50:         readableDuration(summary.Histogram.Percentile(0.50)),
			P99:         readableDuration(summary.Histogram.Percentile(0.99)),
			RowsRead:    readableCount(summary.RowsRead),
			RowsWritten: readableCount(summary.RowsWritten),
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
		return trimZeros(milliseconds/MillisecondsInSecond) + SecondSuffix
	}

	return trimZeros(milliseconds) + MillisecondSuffix
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
