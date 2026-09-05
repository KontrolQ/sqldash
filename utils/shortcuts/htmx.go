package shortcuts

import "github.com/gofiber/fiber/v3"

func isHtmxRequest(context fiber.Ctx) bool {
	return context.Get(HeaderHtmxRequest) == "true" && context.Get(HeaderHtmxBoosted) != "true"
}
