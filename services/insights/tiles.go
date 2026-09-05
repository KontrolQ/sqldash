package insights

import (
	"fmt"
	"strings"

	"sqldash/analytics"
)

type tileSpec struct {
	Label  string
	Value  string
	Now    int64
	Before int64
	Better string
	Alarm  bool
	Pick   func(analytics.Point) float64
}

func tilesFor(now *analytics.Summary, before *analytics.Summary, points []analytics.Point) []TileView {
	specs := []tileSpec{
		{
			Label: QueriesLabel, Value: readableCount(now.Count),
			Now: now.Count, Before: before.Count, Better: MoreIsBetter,
			Pick: func(point analytics.Point) float64 { return float64(point.Total) },
		},
		{
			Label: FailuresLabel, Value: readableCount(now.Failures),
			Now: now.Failures, Before: before.Failures, Better: LessIsBetter, Alarm: now.Failures > 0,
			Pick: func(point analytics.Point) float64 { return float64(point.Failures) },
		},
		{
			Label: RowsReadLabel, Value: readableCount(now.RowsRead),
			Now: now.RowsRead, Before: before.RowsRead, Better: NeitherIsBetter,
			Pick: func(point analytics.Point) float64 { return float64(point.Reads) },
		},
		{
			Label: RowsWrittenLabel, Value: readableCount(now.RowsWritten),
			Now: now.RowsWritten, Before: before.RowsWritten, Better: NeitherIsBetter,
			Pick: func(point analytics.Point) float64 { return float64(point.Writes) },
		},
		{
			Label: RowsReturnedLabel, Value: readableCount(now.RowsReturned),
			Now: now.RowsReturned, Before: before.RowsReturned, Better: NeitherIsBetter,
			Pick: func(point analytics.Point) float64 { return float64(point.Reads) },
		},
		{
			Label: TransferredLabel, Value: readableBytes(now.BytesIn + now.BytesOut),
			Now: now.BytesIn + now.BytesOut, Before: before.BytesIn + before.BytesOut,
			Better: NeitherIsBetter,
			Pick:   func(point analytics.Point) float64 { return float64(point.Bytes) },
		},
	}

	tiles := make([]TileView, 0, len(specs))

	for _, spec := range specs {
		tiles = append(tiles, measure(spec, points))
	}

	return tiles
}

func measure(spec tileSpec, points []analytics.Point) TileView {
	change, kind := readableChange(spec.Now, spec.Before)
	line := spark(points, spec.Pick)

	return TileView{
		Label:  spec.Label,
		Value:  spec.Value,
		Change: change,
		Kind:   kind,
		Tone:   toneOf(kind, spec.Better),
		Alarm:  spec.Alarm,
		Spark:  line,
		Fill:   closed(line),
	}
}

func closed(line string) string {
	if line == "" {
		return ""
	}

	return fmt.Sprintf(FillFormat, line, SparkWidth, SparkHeight, SparkHeight)
}

func toneOf(kind string, better string) string {
	if kind == ChangeSteady || better == NeitherIsBetter {
		return PlainTone
	}

	if (kind == ChangeUp && better == MoreIsBetter) || (kind == ChangeDown && better == LessIsBetter) {
		return GoodTone
	}

	return BadTone
}

func scaled(milliseconds float64) int64 {
	return int64(milliseconds * DurationScale)
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
