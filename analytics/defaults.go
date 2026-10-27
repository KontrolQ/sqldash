package analytics

import "time"

const (
	LogPrefix = "Analytics"

	BucketCount      = 40
	FirstBoundaryMs  = 0.0625
	BucketsPerOctave = 2.0

	RollInterval  = 30 * time.Second
	PruneInterval = time.Hour

	FoldBatch = 5000
)

const (
	EmptyHistogram = "[]"
)

const (
	FoldedMarker = "folded-statement"
)

const (
	SeriesSlots = 60
)
