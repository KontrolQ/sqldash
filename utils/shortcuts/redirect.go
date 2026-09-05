package shortcuts

import (
	"sqldash/utils/collections"
	"sqldash/utils/urls"

	"github.com/gofiber/fiber/v3"
)

func Redirect(context fiber.Ctx, routeName string) error {
	fullPath, exists := urls.GetFullPath(routeName)
	if !exists {
		return fiber.ErrNotFound
	}

	return RedirectToPath(context, fullPath)
}

func RedirectToPath(context fiber.Ctx, path string) error {
	if isHtmxRequest(context) {
		return forward(context, path)
	}

	return context.Redirect().Status(fiber.StatusSeeOther).To(path)
}

func RedirectRoute(context fiber.Ctx, routeName string, params collections.Record[string, string]) error {
	fullPath, exists := urls.ResolvePath(routeName, params)
	if !exists {
		return fiber.ErrNotFound
	}

	return RedirectToPath(context, fullPath)
}

func RedirectFull(context fiber.Ctx, routeName string) error {
	fullPath, exists := urls.GetFullPath(routeName)
	if !exists {
		return fiber.ErrNotFound
	}

	if isHtmxRequest(context) {
		context.Set(HeaderHtmxRedirect, fullPath)
		return context.SendStatus(fiber.StatusNoContent)
	}

	return context.Redirect().Status(fiber.StatusSeeOther).To(fullPath)
}

func RedirectExternal(context fiber.Ctx, address string) error {
	return context.Redirect().Status(fiber.StatusSeeOther).To(address)
}

func forward(context fiber.Ctx, path string) error {
	requestCtx := context.RequestCtx()

	requestCtx.Request.URI().SetPath(path)
	requestCtx.Request.Header.SetMethod(fiber.MethodGet)
	context.App().Handler()(requestCtx)

	return nil
}
