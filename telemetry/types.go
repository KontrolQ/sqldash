package telemetry

import "time"

type Observation struct {
	DatabaseName string
	SQL          string
	DurationMs   float64
	RowsRead     int64
	RowsWritten  int64
	RowsReturned int64
	BytesIn      int64
	BytesOut     int64
	Failed       bool
	OccurredAt   time.Time
}
