package databases

const (
	ListFailedLog   = "Failed to list databases: %v"
	CreateFailedLog = "Failed to create database %s: %v"
	DeleteFailedLog = "Failed to delete database %s: %v"
	RegisterLog     = "Failed to register database %s: %v"
)

const (
	ListUnavailable       = "The list of databases could not be read."
	NameUnusable          = "A name must be lowercase letters, digits and hyphens, and cannot start or end with a hyphen."
	NameTooShort          = "That name is too short."
	NameTooLong           = "That name is too long."
	NameReserved          = "That name is reserved."
	NameTaken             = "A database with that name already exists."
	NameAllocationRefused = "An available database name could not be allocated. Try again."
	CreateRefused         = "The database could not be created."
	DeleteRefused         = "The database could not be deleted."
	DatabaseMissing       = "No such database."
	DatabaseHeld          = "That database is protected. Remove the protection before deleting it."
)

const (
	StatisticsFailedLog    = "Failed to read statistics for %s: %v"
	ConfigurationFailedLog = "Failed to read or save settings for %s: %v"
)

const (
	SettingsRefused = "Those settings could not be saved."
)

const (
	ImportFailedLog = "Failed to import %s: %v"
	ForkFailedLog   = "Failed to fork %s: %v"
	ExportFailedLog = "Failed to export %s: %v"
)

const (
	ImportRefused   = "That file could not be loaded."
	ForkRefused     = "The copy could not be made."
	ExportRefused   = "The database could not be exported."
	OnlyDumpsFormat = "only a .sql dump can be loaded, not %s"
	MomentUnusable  = "That is not a moment this database can be restored to."
	MomentAhead     = "A database cannot be restored to a moment that has not happened yet."
	OnlyKnownFormat = "only a .sql dump or a SQLite file can be loaded, not %s"

	ImportRefusedFormat    = "That file could not be loaded: %v."
	StatementRefusedFormat = "the server refused %q: %w"
	NothingToReplay        = "the file holds no statements"
)

const (
	RuleFailedLog = "Failed to change the access rules for %s: %v"
)

const (
	NetworkUnusable = "That is not an address range. Write it in CIDR form, such as 203.0.113.0/24."
	NetworkRefused  = "That range could not be saved."
	NetworkMissing  = "No such range."
	NetworkAllowed  = "That range may now connect."
	NetworkRemoved  = "That range was removed."
)

const (
	AuthorityRefused             = "You no longer have permission to perform that action."
	EngineRefused                = "That database engine or operation is not supported."
	PostgresUnavailable          = "PostgreSQL is not available on this instance."
	RedisUnavailable             = "Redis is not available on this instance."
	RedisMemoryAllocationRefused = "Redis memory allocation was refused. The installation budget is %s, including %s reserved per database; the limit must also cover current usage. Increase SQLDASH_REDIS_MEMORY_BUDGET if a larger allocation is needed."
	DatabaseNotReady             = "This database is not ready. Its operation is pending or needs recovery."
	OperationFailed              = "The operation could not be completed. The database remains unavailable until recovery or deletion completes."
	SettingsUnavailable          = "The current database settings could not be read."
	ExtensionsAuthorityRefused   = "Copying or restoring installed extensions requires permission to manage PostgreSQL extensions."
	RemoteRefused                = "The remote PostgreSQL database could not be inspected or copied. Check its address, credentials and TLS settings."
	RemoteConnectionRefused      = "Could not connect to the source PostgreSQL database. Check its hostname, port, credentials and TLS settings."
	ExtensionsMustBeManaged      = "This source requires PostgreSQL extensions. Importing or cloning extension-dependent databases is not supported. Extensions can only be enabled after a database has been created."
)
