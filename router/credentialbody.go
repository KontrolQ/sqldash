package router

import (
	"io"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const maximumCredentialBodyBytes = 64 * 1024

func prepareCredentialBody(ctx fiber.Ctx, name string) error {
	if !strings.HasPrefix(name, "auth.") || ctx.Method() == fiber.MethodGet || ctx.Method() == fiber.MethodHead {
		return nil
	}
	request := ctx.Request()
	if request.Header.ContentLength() > maximumCredentialBodyBytes {
		ctx.Response().Header.SetConnectionClose()
		return fiber.ErrRequestEntityTooLarge
	}
	if request.IsBodyStream() {
		contents, err := io.ReadAll(io.LimitReader(request.BodyStream(), maximumCredentialBodyBytes+1))
		if err != nil || len(contents) > maximumCredentialBodyBytes {
			ctx.Response().Header.SetConnectionClose()
			return fiber.ErrRequestEntityTooLarge
		}
		request.SetBodyRaw(contents)
	} else if len(request.Body()) > maximumCredentialBodyBytes {
		return fiber.ErrRequestEntityTooLarge
	}
	return nil
}
