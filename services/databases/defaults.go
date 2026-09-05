package databases

const (
	LogPrefix = "Databases"

	IndexTitle = "Databases"

	MinimumNameLength = 2
	MaximumNameLength = 48
	NamePattern       = `^[a-z0-9][a-z0-9-]*[a-z0-9]$`
)

var ReservedNames = []string{"sqldash", "admin", "api", "static", "login", "setup", "logout"}

const (
	AddressScheme = "https://"

	UnknownSize  = "unknown"
	BytesFormat  = "%d B"
	SizeFormat   = "%.1f %s"
	KilobyteSize = 1024
	KilobyteUnit = "KB"
	MegabyteUnit = "MB"
	GigabyteUnit = "GB"
	TerabyteUnit = "TB"
)

const (
	StagedNameFormat = "%s-%d.sql"
	DumpSuffix       = ".sql"
	ExportSuffix     = ".sql"
	ForkPrefix       = "restored-"
	MomentLayout     = "2006-01-02T15:04"
)

const (
	ExportPageSize = 500
	TableKind      = "table"
	StatementEnd   = ";\n"
	NullLiteral    = "NULL"
	BlobFormat     = "X'%X'"
	InsertFormat   = "INSERT INTO %s (%s) VALUES (%s);\n"
	DumpHeader     = "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\n"
	DumpFooter     = "COMMIT;\n"

	SchemaSQL = `SELECT type, name, sql FROM sqlite_master
		WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%'
		ORDER BY CASE type WHEN 'table' THEN 0 WHEN 'view' THEN 1 ELSE 2 END, name`

	PageSQL = "SELECT * FROM %s LIMIT %d OFFSET %d"
)
