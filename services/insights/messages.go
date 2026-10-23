package insights

const (
	ReadFailedLog = "Failed to read measurements: %v"
)

const (
	MeasurementsUnavailable = "The measurements could not be read."
	PostgresUnavailable     = "PostgreSQL is not enabled."
	EngineAuthorityRefused  = "You cannot view insights for this database engine."

	NoPostgresDatabasesTitle = "No accessible PostgreSQL databases"
	NoPostgresDatabasesNote  = "PostgreSQL insights are limited to databases you can monitor."
	PostgresNotReadyTitle    = "PostgreSQL databases are unavailable"
	PostgresNotReadyNote     = "Statistics become available once an accessible PostgreSQL database is ready."
	PostgresWaitingTitle     = "Waiting for PostgreSQL statistics"
	PostgresWaitingNote      = "No samples have been collected for accessible PostgreSQL databases."
	PostgresStaleTitle       = "PostgreSQL statistics are unavailable"
	PostgresStaleNote        = "No recent sample is available. The last collected snapshots are shown below."
	StatementsDisabledTitle  = "Query statistics are not enabled"
	StatementsDisabledNote   = "Enable pg_stat_statements in Extensions to collect query statistics. Database size and connection counts remain available."
	StatementsWaitingTitle   = "Waiting for query statistics"
	StatementsWaitingNote    = "At least two compatible samples are needed to measure query activity. Restarts and resets can leave gaps in this period."
	NoPostgresQueriesTitle   = "No queries recorded"
	NoPostgresQueriesNote    = "No completed top-level statements were measured in this period."
)
