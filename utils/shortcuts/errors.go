package shortcuts

import (
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

func RouteError(context fiber.Ctx, routeError *fiber.Error) error {
	meta.SetPageTitle(context, routeError.Error())
	return RenderWithStatus(context, TemplateError, routeError, routeError.Code)
}

func ServiceError(code int, message string) *fiber.Error {
	return fiber.NewError(code, message)
}
