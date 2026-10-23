package main

import (
	"context"
	"os"
	"sqldash/database"
	_ "sqldash/engines/postgres"
	"time"

	"sqldash/utils/logger"
)

func main() {
	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	if mode == "health" {
		runHealth()
		return
	}
	initializationContext, cancelInitialization := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelInitialization()
	var initializationError error
	if os.Getenv("SQLDASH_SCHEMA_VERIFIED") == "1" {
		initializationError = database.Verify()
	} else {
		initializationError = database.Initialize(initializationContext)
	}
	if initializationError != nil {
		logger.Fatalf(LogPrefix, "Could not initialize application state: %v", initializationError)
	}

	switch mode {
	case "":
		supervise()
	case ModeProxy:
		runProxy()
	case ModeWeb:
		runWeb()
	case ModePostgresGateway:
		runPostgresGateway()
	default:
		logger.Fatalf(LogPrefix, UnknownModeLog, mode, ModeProxy, ModeWeb)
	}
}
