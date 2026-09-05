package insights

import (
	"fmt"
	"strings"

	"sqldash/analytics"
)

func tilesFor(now *analytics.Summary, before *analytics.Summary, points []analytics.Point) []TileView {
	return []TileView{
		measure(QueriesLabel, readableCount(now.Count), now.Count, before.Count, spark(points, func(point analytics.Point) float64 {
			return float64(point.Total)
		})),
		measure(FailuresLabel, readableCount(now.Failures), now.Failures, before.Failures, spark(points, func(point analytics.Point) float64 {
			return float64(point.Failures)
		})),
		measure(RowsReadLabel, readableCount(now.RowsRead), now.RowsRead, before.RowsRead, spark(points, func(point analytics.Point) float64 {
			return float64(point.Total)
		})),
		measure(RowsWrittenLabel, readableCount(now.RowsWritten), now.RowsWritten, before.RowsWritten, spark(points, func(point analytics.Point) float64 {
			return float64(point.Writes)
		})),
		measure(AverageLabel, readableDuration(now.Average()), int64(now.Average()*100), int64(before.Average()*100), spark(points, func(point analytics.Point) float64 {
			return point.P50
		})),
		measure(TailLabel, readableDuration(now.Histogram.Percentile(0.99)), int64(now.Histogram.Percentile(0.99)*100), int64(before.Histogram.Percentile(0.99)*100), spark(points, func(point analytics.Point) float64 {
			return point.P99
		})),
	}
}

func measure(label string, value string, now int64, before int64, path string) TileView {
	change, kind := readableChange(now, before)

	return TileView{
		Label:  label,
		Value:  value,
		Change: change,
		Kind:   kind,
		Spark:  path,
	}
}

func spark(points []analytics.Point, pick func(analytics.Point) float64) string {
	if len(points) < 2 {
		return ""
	}

	highest := 0.0

	for _, point := range points {
		if value := pick(point); value > highest {
			highest = value
		}
	}

	if highest == 0 {
		return ""
	}

	step := SparkWidth / float64(len(points)-1)
	pieces := make([]string, 0, len(points))

	for index, point := range points {
		across := float64(index) * step
		down := SparkHeight - pick(point)/highest*(SparkHeight-SparkPadding)

		marker := LineTo
		if index == 0 {
			marker = MoveTo
		}

		pieces = append(pieces, fmt.Sprintf(PointFormat, marker, across, down))
	}

	return strings.TrimSpace(strings.Join(pieces, ""))
}
