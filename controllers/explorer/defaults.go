package explorer

const (
	LogPrefix = "ExplorerController"

	ExplorePath   = "/databases/"
	BrowseSuffix  = "/explore"
	ConsoleSuffix = "/console"

	NameParameter   = "name"
	CheckedValue    = "on"
	ConsoleTemplate = "explorer/console"

	ColumnPrefix    = "column."
	ColumnSeparator = ","
)

const (
	OneChangedFormat = "One cell in %s"
	ChangedFormat    = "%d cells in %s"
	OneDeletedFormat = "One row from %s"
	DeletedFormat    = "%d rows from %s"
	InsertedFormat   = "Added to %s"
	StatementLength  = 120
	Ellipsis         = "…"
)
