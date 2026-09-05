package explorer

const (
	LogPrefix = "Explorer"

	DefaultPageSize = 50
	MaximumPageSize = 200
	MaximumCellText = 160
	RowIdentifier   = "rowid"

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
