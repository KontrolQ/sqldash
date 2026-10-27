package pictures

const (
	LogPrefix = "Pictures"

	MaximumSide          = 512
	MaximumDecodedPixels = 40000000
	MaximumFrames        = 200
	HistoryLimit         = 5
	JPEGQuality          = 85
	MaximumStoredBytes   = 256 << 20
	ReferencePrefix      = "file:"
	URLPrefix            = "/pictures/"
	DirectoryPermissions = 0700
	FilePermissions      = 0600
)
