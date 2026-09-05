package telemetry

const (
	QueueFullLog   = "Dropped a statement because the telemetry queue is full"
	WriteFailedLog = "Failed to record %d statements: %v"
	FingerprintLog = "Failed to record a fingerprint for %s: %v"
)
