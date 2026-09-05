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

	UnknownSize  = "unknown"
	BytesFormat  = "%d B"
	SizeFormat   = "%.1f %s"
	KilobyteSize = 1024
	KilobyteUnit = "KB"
	MegabyteUnit = "MB"
	GigabyteUnit = "GB"
	TerabyteUnit = "TB"
)
