package main

import (
	"os"

	"sqldash/utils/logger"
)

func main() {
	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "":
		supervise()
	case ModeProxy:
		runProxy()
	case ModeWeb:
		runWeb()
	default:
		logger.Fatalf(LogPrefix, UnknownModeLog, mode, ModeProxy, ModeWeb)
	}
}
