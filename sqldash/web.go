package main

import (
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
		ErrorHandler: router.ErrorHandler,
		Views:        engine,
	})

	middleware.Initialize(application)
	router.Initialize(application)

	logger.Successf(LogPrefix, StartedLog, config.Server.WebAddress)

	if listenError := application.Listen(config.Server.WebAddress, fiber.ListenConfig{
		DisableStartupMessage: true,
	}); listenError != nil {
		logger.Fatalf(LogPrefix, ListenFailedLog, listenError)
	}
}
