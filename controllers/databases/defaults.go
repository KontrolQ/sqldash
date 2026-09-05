package databases

const (
	LogPrefix = "DatabasesController"

	IndexRoute = "databases"
	ShowRoute  = "databases.show"
	IndexPath  = "/"
	ShowPath   = "/databases/"

	NameParameter = "name"
	CheckedValue  = "on"

	DumpField         = "dump"
	DispositionHeader = "content-disposition"
	ContentTypeHeader = "content-type"
	DumpContentType   = "application/sql"
)
