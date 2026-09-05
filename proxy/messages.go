package proxy

const (
	StartedLog         = "Proxy listening on %s"
	ListenFailedLog    = "Failed to listen on %s: %v"
	UnknownDatabaseLog = "Refused a request for unknown database %q"
	ForwardFailedLog   = "Failed to forward to the database server: %v"
)

const (
	UnknownDatabase = "No such database."
	DashboardDown   = "The dashboard is not answering."
	DatabaseDown    = "The database server is not answering."
)
