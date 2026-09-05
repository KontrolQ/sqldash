package explorer

const (
	ReadFailedLog = "Failed to read from %s: %v"
)

const (
	TablesUnavailable = "The tables could not be read."
	RowsUnavailable   = "The rows could not be read."
	TableMissing      = "No such table."
	NotEditable       = "This table has no primary key and no rowid, so its rows cannot be aimed at."
	StatementEmpty    = "Write a statement to run."
)

const (
	WriteFailedLog = "Failed to write to %s: %v"
)

const (
	ColumnMissing = "No such column."
	CellSaved     = "Saved."
	RowsDeleted   = "%d rows were deleted."
	NothingChosen = "Pick at least one row first."
)

const (
	NothingToInsert = "Fill in at least one column."
	RowAdded        = "That row was added."
)
