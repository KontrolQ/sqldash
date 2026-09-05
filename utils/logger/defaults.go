package logger

const (
	AnsiReset = "\033[0m"
)

const (
	LevelColorDebug = "\033[35mDEBUG  \033[0m"
	LevelColorError = "\033[31mERROR  \033[0m"
	LevelColorInfo  = "\033[34mINFO   \033[0m"
	LevelColorWarn  = "\033[33mWARN   \033[0m"
)

const (
	MessageColorDebug   = "\033[90m"
	MessageColorError   = "\033[31m"
	MessageColorInfo    = "\033[97m"
	MessageColorSuccess = "\033[32m"
	MessageColorWarn    = "\033[33m"
)

const (
	PrefixColor = "\033[36m"
	PrefixWidth = 18
)
