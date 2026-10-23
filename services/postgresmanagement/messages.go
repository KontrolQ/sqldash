package postgresmanagement

const (
	PostgresRequired            = "This operation requires a PostgreSQL database."
	ActivityUnavailable         = "PostgreSQL activity could not be read."
	ActivityRefused             = "The connection has changed or is not a managed application connection."
	ActivityFailed              = "The connection operation could not be confirmed."
	ExtensionsUnavailable       = "PostgreSQL extensions could not be read."
	ExtensionRefused            = "Only available, supported extensions can be enabled or disabled. Required extensions cannot be changed."
	ExtensionFailed             = "The extension could not be changed. Disabling is refused while database objects depend on it."
	ExtensionDependencyRequired = "Enable %s in Extensions first."
)
