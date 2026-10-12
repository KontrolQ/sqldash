package sessions

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func Version(context fiber.Ctx) uint {
	held := session.FromContext(context)
	if held == nil {
		return 0
	}
	version, _ := held.Get("session-version").(uint)
	return version
}
