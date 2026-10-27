package database

import "time"

const (
	LogPrefix         = "Database"
	SchemaReadyMarker = "database-permission-namespaces"

	SlowQueryThreshold    = 200 * time.Millisecond
	MaxConnectionLifetime = time.Hour
	MaxIdleConnections    = 5
	MaxOpenConnections    = 25
)

var StartupPragmas = []string{
	"PRAGMA journal_mode = WAL",
	"PRAGMA synchronous = NORMAL",
	"PRAGMA busy_timeout = 5000",
	"PRAGMA foreign_keys = ON",
}
