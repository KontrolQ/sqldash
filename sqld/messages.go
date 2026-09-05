package sqld

const (
	EncodeFailed  = "The request to the database server could not be encoded."
	BuildFailed   = "The request to the database server could not be built."
	SendFailed    = "The database server could not be reached."
	ReadFailed    = "The answer from the database server could not be read."
	DecodeFailed  = "The answer from the database server could not be understood."
	RefusedFormat = "The database server refused the request: %s %s"
)

const (
	QueryRefused = "The database refused that query."
)
