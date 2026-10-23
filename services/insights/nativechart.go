package insights

import (
	"fmt"
	"strings"

	"sqldash/pgmetrics"
)

type nativeChartSpec struct {
	Label      string
	Unit       string
	Axis       string
	Value      string
	Pick       func(pgmetrics.SeriesPoint) *float64
	Statements bool
}

func applyNativeCharts(shown *NativeContext, series *pgmetrics.Series) {
	calls, execution, storage, connections := UnavailableValue, UnavailableValue, UnavailableValue, UnavailableValue
	if series.RecordedCalls != nil {
		calls = readableCount(*series.RecordedCalls)
	}
	if series.RecordedExecution != nil {
		execution = readableDuration(*series.RecordedExecution)
	}
	if series.LatestStorageBytes != nil {
		storage = readableBytes(*series.LatestStorageBytes)
	}
	if series.LatestConnections != nil {
		connections = readableCount(*series.LatestConnections)
	}
	specifications := []nativeChartSpec{
		{Label: "Recorded calls", Axis: "calls", Value: calls, Statements: true, Pick: func(point pgmetrics.SeriesPoint) *float64 { return nativeChartNumber(point.Calls) }},
		{Label: "Recorded execution time", Unit: MillisecondUnit, Value: execution, Statements: true, Pick: func(point pgmetrics.SeriesPoint) *float64 { return point.ExecutionMs }},
		{Label: "Database storage", Unit: "bytes", Value: storage, Pick: func(point pgmetrics.SeriesPoint) *float64 { return nativeChartNumber(point.StorageBytes) }},
		{Label: "Connections", Axis: "connections", Value: connections, Pick: func(point pgmetrics.SeriesPoint) *float64 { return nativeChartNumber(point.Connections) }},
	}
	charts := make([]string, 0, len(specifications))
	shown.Tiles = make([]TileView, 0, len(specifications))
	for _, specification := range specifications {
		chart := chartData{Kind: LineKind, Unit: specification.Unit, Axis: specification.Axis, Series: []string{specification.Label}, Table: true}
		measured := false
		for _, point := range series.Points {
			value := specification.Pick(point)
			measured = measured || value != nil
			note := ""
			if specification.Statements && point.CounterPartial {
				note = "Partial statement coverage: only compatible recorded deltas are included; unknown statement baselines are excluded."
			}
			chart.Points = append(chart.Points, chartPoint{Label: point.At.Local().Format(MomentLayout), Values: []*float64{value}, BreakBefore: point.BreakBefore, Note: note})
		}
		encoded := ""
		if measured {
			encoded = encode(chart)
		}
		charts = append(charts, encoded)
		line, fill := nativeSpark(chart.Points)
		quiet := "No trend yet"
		if specification.Value == UnavailableValue {
			quiet = "No complete measurement"
		}
		shown.Tiles = append(shown.Tiles, TileView{Label: specification.Label, Value: specification.Value, Spark: line, Fill: fill, Quiet: quiet})
	}
	shown.CallsChart, shown.ExecutionChart, shown.StorageChart, shown.ConnectionsChart = charts[0], charts[1], charts[2], charts[3]
	shown.ChartCoverage = fmt.Sprintf("Statement samples cover %d of %d buckets across all %d selected PostgreSQL databases; %d of those buckets have partial statement coverage because unknown baselines are excluded. Database samples cover %d buckets. Gaps are unavailable, not zero. Recorded totals above sum only compatible recorded deltas, not all client activity. Latest storage and connection totals require fresh samples from every selected database.", series.CounterBuckets, len(series.Points), series.DatabaseCount, series.PartialCounterBuckets, series.GaugeBuckets)
}

func nativeChartNumber(value *int64) *float64 {
	if value == nil {
		return nil
	}
	number := float64(*value)
	return &number
}

func nativeSpark(points []chartPoint) (string, string) {
	if len(points) < 2 {
		return "", ""
	}
	highest := 1.0
	for _, point := range points {
		if len(point.Values) > 0 && point.Values[0] != nil && *point.Values[0] > highest {
			highest = *point.Values[0]
		}
	}
	step := SparkWidth / float64(len(points)-1)
	var line, fill strings.Builder
	var segment strings.Builder
	firstAcross, lastAcross := 0.0, 0.0
	length := 0
	finish := func() {
		if length > 1 {
			line.WriteString(segment.String())
			fmt.Fprintf(&fill, "%s L %.2f %.2f L %.2f %.2f Z ", segment.String(), lastAcross, SparkHeight, firstAcross, SparkHeight)
		}
		segment.Reset()
		length = 0
	}
	for index, point := range points {
		if point.BreakBefore {
			finish()
		}
		if len(point.Values) == 0 || point.Values[0] == nil {
			finish()
			continue
		}
		across := float64(index) * step
		down := SparkHeight - SparkPadding - *point.Values[0]/highest*(SparkHeight-2*SparkPadding)
		marker := LineTo
		if length == 0 {
			firstAcross = across
			marker = MoveTo
		}
		fmt.Fprintf(&segment, PointFormat, marker, across, down)
		lastAcross = across
		length++
	}
	finish()
	return strings.TrimSpace(line.String()), strings.TrimSpace(fill.String())
}
