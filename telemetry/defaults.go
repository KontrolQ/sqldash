package telemetry

import "time"

const (
	LogPrefix = "Telemetry"

	QueueDepth    = 4096
	BatchSize     = 256
	FlushInterval = 2 * time.Second
	DigestLength  = 16
)
