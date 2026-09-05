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

	Thousand = 1_000.0
	Million  = 1_000_000.0
	Billion  = 1_000_000_000.0

	ThousandSuffix = "k"
	MillionSuffix  = "M"
	BillionSuffix  = "B"

	CompactFormat = "%.2f"
	NoNumber      = "0"

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
	FileNameFormat   = "%s-%d.db"
	DumpSuffix       = ".sql"
	ExportSuffix     = ".sql"
	FileSuffix       = ".db"

	SQLiteMagic    = "SQLite format 3\x00"
	SQLiteDriver   = "sqlite"
	ReadOnlySource = "file:"
	ReadBufferSize = 1 << 16
	ReplayBatch    = 128
	ReplayGuard    = "PRAGMA foreign_keys=OFF"
	CommentMarker  = "--"
	SequenceTable  = "SQLITE_SEQUENCE"
	SnippetLength  = 80
	Ellipsis       = "…"

	EveryRowSQL        = "SELECT * FROM %s"
	ReplayFailedFormat = "%s: %w"
	ForkPrefix         = "restored-"
	MomentLayout       = "2006-01-02T15:04"
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

const (
	ReadWriteValue = "read-write"
	ReadWriteLabel = "Read and write"
	ReadOnlyValue  = "read-only"
	ReadOnlyLabel  = "Read only"
)

var SkippedPrefixes = []string{"BEGIN", "COMMIT", "ROLLBACK", "PRAGMA", "VACUUM", "ANALYZE"}
