package audit

const (
	RecordFailedLog = "Failed to record %s: %v"
	ReadFailedLog   = "Failed to read the audit log: %v"
)

const (
	LogUnavailable              = "The audit log could not be read."
	InvalidFilter               = "The audit log filter is invalid."
	DatabaseIdentityUnavailable = "The database identity could not be found."
)
