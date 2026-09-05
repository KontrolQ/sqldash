package databases

const (
	LogPrefix = "DatabasesController"

	IndexRoute = "databases"
	ShowRoute  = "databases.show"
	IndexPath  = "/"
	ShowPath   = "/databases/"

	NameParameter = "name"
	RuleParameter = "rule"
	CheckedValue  = "on"

	DumpField         = "dump"
	DispositionHeader = "content-disposition"
	ContentTypeHeader = "content-type"
	FileContentType   = "application/vnd.sqlite3"
	DumpContentType   = "application/sql"
)

const (
	StartedEmpty   = "Started empty"
	LoadedFrom     = "Loaded from %s"
	CopiedFrom     = "Copied from %s"
	CopiedAsAtWhen = "Copied from %s as it stood at %s"
	RangeAllowed   = "Allowed %s"
	RangeRemoved   = "Removed %s"
	SQLiteFileKind = "SQLite file"
	DumpKind       = "SQL dump"
	BlockedReads   = "reads blocked"
	BlockedWrites  = "writes blocked"
	AllowedAttach  = "ATTACH allowed"
	Protected      = "delete protection on"
	NothingBlocked = "nothing blocked"
)
