package explorer

const (
	LogPrefix = "Explorer"

	DefaultPageSize = 50
	MaximumPageSize = 200
	MaximumCellText = 160
	MaximumPeeks    = 60
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
	EmptySchema  = "[]"

	SchemaSQL = `SELECT m.name, p.name FROM sqlite_master m
		JOIN pragma_table_info(m.name) p
		WHERE m.type IN ('table','view') AND m.name NOT LIKE 'sqlite_%'
		ORDER BY m.name, p.cid`

	TableKey     = "table"
	SearchKey    = "search"
	ColumnKey    = "column"
	OperatorKey  = "operator"
	ValueKey     = "value"
	PageKey      = "page"
	SortKey      = "sort"
	DirectionKey = "direction"

	OperatorContains = "contains"
	OperatorMisses   = "misses"
	OperatorIs       = "is"
	OperatorIsNot    = "isnot"
	OperatorStarts   = "starts"
	OperatorEnds     = "ends"
	OperatorAbove    = "above"
	OperatorAtLeast  = "atleast"
	OperatorBelow    = "below"
	OperatorAtMost   = "atmost"
	OperatorEmpty    = "empty"
	OperatorFilled   = "filled"

	CreatedOrigin    = "c"
	UniqueOrigin     = "u"
	PrimaryKeyOrigin = "pk"

	CreatedLabel    = "declared"
	UniqueLabel     = "unique constraint"
	PrimaryKeyLabel = "primary key"

	ContainsLabel = "contains"
	MissesLabel   = "does not contain"
	IsLabel       = "is"
	IsNotLabel    = "is not"
	StartsLabel   = "starts with"
	EndsLabel     = "ends with"
	AboveLabel    = "is greater than"
	AtLeastLabel  = "is at least"
	BelowLabel    = "is less than"
	AtMostLabel   = "is at most"
	EmptyLabel    = "is empty"
	FilledLabel   = "is not empty"
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
