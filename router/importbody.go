package router

import (
	"errors"

	"sqldash/config"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

func prepareImportBody(ctx fiber.Ctx) error {
	maximumBody := 0
	if config.Transfer.MaximumBytes > 0 {
		maximumBody = int(config.Transfer.MaximumBytes) + config.MultipartUploadOverhead
	}
	form, err := ctx.Request().MultipartFormWithLimit(maximumBody)
	if err != nil {
		ctx.Response().Header.SetConnectionClose()
		if errors.Is(err, fasthttp.ErrBodyTooLarge) {
			return fiber.ErrRequestEntityTooLarge
		}
		return fiber.ErrBadRequest
	}
	if config.Transfer.MaximumBytes > 0 {
		for _, files := range form.File {
			for _, file := range files {
				if file.Size > config.Transfer.MaximumBytes {
					return fiber.ErrRequestEntityTooLarge
				}
			}
		}
	}
	return nil
}
