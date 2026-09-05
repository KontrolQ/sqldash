package insights

import (
	"encoding/json"

	"sqldash/analytics"
)

type chartPoint struct {
	Label  string    `json:"label"`
	Values []float64 `json:"values"`
}

type chartData struct {
	Kind   string       `json:"kind"`
	Unit   string       `json:"unit"`
	Axis   string       `json:"axis"`
	Series []string     `json:"series"`
	Points []chartPoint `json:"points"`
}

func trafficChart(points []analytics.Point) string {
	held := chartData{Kind: StackedKind, Axis: RowsAxis, Series: []string{ReadsName, WritesName}}

	measured := false

	for _, point := range points {
		if point.Total > 0 {
			measured = true
		}

		held.Points = append(held.Points, chartPoint{
			Label:  point.At.Local().Format(MomentLayout),
			Values: []float64{float64(point.Reads), float64(point.Writes)},
		})
	}

	if !measured {
		return ""
	}

	return encode(held)
}

func latencyChart(points []analytics.Point) string {
	held := chartData{Kind: LineKind, Unit: MillisecondUnit, Series: []string{P50Name, P95Name, P99Name}}

	measured := false

	for _, point := range points {
		if point.P99 > 0 {
			measured = true
		}

		held.Points = append(held.Points, chartPoint{
			Label:  point.At.Local().Format(MomentLayout),
			Values: []float64{point.P50, point.P95, point.P99},
		})
	}

	if !measured {
		return ""
	}

	return encode(held)
}

func encode(held chartData) string {
	encoded, encodeError := json.Marshal(held)
	if encodeError != nil {
		return ""
	}

	return string(encoded)
}
