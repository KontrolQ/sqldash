package proxy

const (
	StartedLog           = "Proxy listening on %s"
	ListenFailedLog      = "Failed to listen on %s: %v"
	UnknownDatabaseLog   = "Refused a request for unknown database %q"
	ForwardFailedLog     = "Failed to forward to the database server: %v"
	TokenLookupFailedLog = "Failed to look up a token: %v"
	TokenNoteFailedLog   = "Failed to record a token being used: %v"
)

const (
	UnknownDatabase  = "No such database."
	DashboardDown    = "The dashboard is not answering."
	DatabaseDown     = "The database server is not answering."
	AccessUnreadable = "Access rules could not be read."
	AddressRefused   = "That address is not allowed to reach this database."
	TokenMissing     = "A token is required."
	TokenRefused     = "That token is not valid for this database."
)
