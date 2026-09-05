package main

const (
	UnknownModeLog   = "Unknown mode %q; run with no argument, or with %q or %q"
	ExecutableLog    = "Failed to find our own executable: %v"
	ListenFailedLog  = "Failed to listen: %v"
	StartedLog       = "Listening on %s"
	ShuttingDownLog  = "Shutting down"
	UnmanagedSqldLog = "Not starting a database server; expecting one at %s"
)
