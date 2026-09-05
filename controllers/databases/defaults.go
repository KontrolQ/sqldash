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
