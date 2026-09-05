package databases

const (
	ListFailedLog   = "Failed to list databases: %v"
	CreateFailedLog = "Failed to create database %s: %v"
	DeleteFailedLog = "Failed to delete database %s: %v"
	RegisterLog     = "Failed to register database %s: %v"
)

const (
	ListUnavailable = "The list of databases could not be read."
	NameUnusable    = "A name must be lowercase letters, digits and hyphens, and cannot start or end with a hyphen."
	NameTooShort    = "That name is too short."
	NameTooLong     = "That name is too long."
	NameReserved    = "That name is reserved."
	NameTaken       = "A database with that name already exists."
	CreateRefused   = "The database could not be created."
	DeleteRefused   = "The database could not be deleted."
	DatabaseMissing = "No such database."
	DatabaseHeld    = "That database is protected. Remove the protection before deleting it."
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
