package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"sqldash/config"
	"sqldash/redishost"
	"sqldash/services/redisconnections"
	"sqldash/utils/logger"
)

func runRedisHost() {
	if !config.Redis.Enabled {
		return
	}
	parent, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := redishost.Run(parent, redisconnections.Reconcile); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatalf(LogPrefix, "Redis runtime stopped: %v", err)
	}
}
