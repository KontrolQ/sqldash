package insights

import (
	"math"
)

type QueryMetricColumn struct {
	Key   string
	Label string
	Help  string
	Path  string
}

type QueryMetricCell struct {
	Value     string
	Reason    string
	Available bool
	Integer   *int64
	Number    *float64
}

type QueryTableRow struct {
	Key            string
	SourceKeys     []string
	Engine         string
	EngineLabel    string
	DatabaseID     uint
	Database       string
	DatabasePath   string
	Statement      string
	FullStatement  string
	StatementLabel string
	StatementID    string
	Cells          []QueryMetricCell
	metrics        map[string]QueryMetricCell
}

func (self QueryTableRow) Metric(key string) QueryMetricCell {
	if cell, exists := self.metrics[key]; exists {
		return cell
	}
	return QueryMetricCell{Value: "Unavailable", Reason: "This measurement is not collected for " + self.EngineLabel + " queries."}
}

func queryIntegerCell(value int64) QueryMetricCell {
	if value < 0 {
		return QueryMetricCell{Value: "Unavailable", Reason: "The recorded counter is invalid."}
	}
	return QueryMetricCell{Value: readableCount(value), Available: true, Integer: &value}
}

func queryNumberCell(value float64, display string) QueryMetricCell {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return QueryMetricCell{Value: "Unavailable", Reason: "The recorded measurement is invalid."}
	}
	return QueryMetricCell{Value: display, Available: true, Number: &value}
}

func queryDurationCell(value float64) QueryMetricCell {
	return queryNumberCell(value, readableDuration(value))
}
