package router

import (
	"sqldash/utils/shortcuts"
	"sqldash/utils/urls"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func Initialize(application *fiber.App) {
	application.Get(StaticPrefix+"/*", static.New(StaticRoot))
	application.Get(RobotsPath, static.New(RobotsFile))
	urls.Attach(application)
}

func ErrorHandler(context fiber.Ctx, routeError error) error {
	held, matched := routeError.(*fiber.Error)
	if !matched {
		held = fiber.NewError(fiber.StatusInternalServerError, routeError.Error())
	}

	return shortcuts.RouteError(context, held)
}
