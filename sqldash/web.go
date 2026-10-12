package main

import (
	"net"

	"sqldash/analytics"
	"sqldash/config"
	"sqldash/middleware"
	"sqldash/router"
	"sqldash/tags"
	"sqldash/utils/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/django/v3"
)

func runWeb() {
	tags.Initialize()
	analytics.Start()

	engine := django.New(TemplateRoot, TemplateExtension)
	engine.Reload(config.Server.Debug)

	application := fiber.New(fiber.Config{
		AppName:      WebProcessName,
		BodyLimit:    config.RequestBodyLimit(),
		ErrorHandler: router.ErrorHandler,
		Views:        engine,
	})

	middleware.Initialize(application)
	router.Initialize(application)

	listener, listenError := net.Listen("tcp4", config.Server.WebAddress)
	if listenError != nil {
		logger.Fatalf(LogPrefix, ListenFailedLog, listenError)
	}
	defer listener.Close()

	logger.Successf(LogPrefix, StartedLog, listener.Addr().String())

	if listenError := application.Listener(listener, fiber.ListenConfig{
		DisableStartupMessage: true,
	}); listenError != nil {
		logger.Fatalf(LogPrefix, ListenFailedLog, listenError)
	}
}
