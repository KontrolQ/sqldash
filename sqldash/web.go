package main

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/django/v3"
	"net"
	"os/signal"
	"sqldash/analytics"
	"sqldash/config"
	"sqldash/middleware"
	"sqldash/pgmetrics"
	"sqldash/router"
	databases "sqldash/services/databases"
	connections "sqldash/services/postgresconnections"
	"sqldash/tags"
	"sqldash/utils/logger"
	"syscall"
	"time"
)

func runWeb() {
	parent, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	tags.Initialize()
	analytics.Start()
	if config.Postgres.Enabled {
		pgmetrics.Start(parent)
	}
	go reconcileOperations(parent)
	engine := django.New(TemplateRoot, TemplateExtension)
	engine.Reload(config.Server.Debug)
	application := fiber.New(fiber.Config{AppName: WebProcessName, BodyLimit: config.RequestBodyLimit(), ErrorHandler: router.ErrorHandler, Views: engine})
	application.Get("/health/ready", readyHealth)
	middleware.Initialize(application)
	router.Initialize(application)
	listener, listenError := net.Listen("tcp4", config.Server.WebAddress)
	if listenError != nil {
		logger.Fatalf(LogPrefix, ListenFailedLog, listenError)
	}
	defer listener.Close()
	go func() {
		<-parent.Done()
		if shutdownError := application.ShutdownWithTimeout(15 * time.Second); shutdownError != nil {
			logger.Errorf(LogPrefix, "Dashboard shutdown could not complete: %v", shutdownError)
		}
	}()
	logger.Successf(LogPrefix, StartedLog, listener.Addr().String())
	if listenError := application.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}); listenError != nil && parent.Err() == nil {
		logger.Fatalf(LogPrefix, ListenFailedLog, listenError)
	}
}

func reconcileOperations(parent context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if reconcileError := databases.Reconcile(parent); reconcileError != nil && parent.Err() == nil {
			logger.Warnf(LogPrefix, "Database operation reconciliation is incomplete")
		}
		if config.Postgres.Enabled {
			if reconcileError := connections.Reconcile(parent); reconcileError != nil && parent.Err() == nil {
				logger.Warnf(LogPrefix, "PostgreSQL credential reconciliation is incomplete")
			}
		}
		select {
		case <-parent.Done():
			return
		case <-ticker.C:
		}
	}
}
