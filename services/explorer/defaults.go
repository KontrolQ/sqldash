package explorer

const (
	LogPrefix = "Explorer"

	DefaultPageSize = 50
	MaximumPageSize = 200
	MaximumCellText = 160
	RowIdentifier   = "rowid"

	DatabasePath    = "/databases/"
	BrowseSuffix    = "/explore"
	StructureSuffix = "/structure"

	AscendingOrder  = "asc"
	DescendingOrder = "desc"

	TableListSQL = `SELECT name, type FROM sqlite_master
		WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%'
		ORDER BY type, name`

	ColumnListSQL     = "PRAGMA table_info(%s)"
	ForeignKeyListSQL = "PRAGMA foreign_key_list(%s)"
	CountSQL          = "SELECT COUNT(*) FROM %s"
	HasRowIdSQL       = "SELECT rowid FROM %s LIMIT 1"

	ConsoleTitle = "Console"
)

const (
	DurationFormat = "%.2f ms"
	BytesFormat    = "%d bytes"
	NullText       = "NULL"
	Ellipsis       = "…"
)

const (
	IndexListSQL  = "PRAGMA index_list(%s)"
	IndexInfoSQL  = "PRAGMA index_info(%s)"
	DefinitionSQL = "SELECT sql FROM sqlite_master WHERE name = ?"
)
