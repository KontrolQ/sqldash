package database

const (
	ConnectionFailedLog = "Failed to open the state database: %v"
	PoolConfigFailedLog = "Failed to configure the connection pool: %v"
	PragmaFailedLog     = "Failed to apply %s: %v"
	MigrationFailedLog  = "Failed to migrate the state database: %v"
	ConnectedLog        = "State database opened at %s"
)
