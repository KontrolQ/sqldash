package sessions

import (
	"sqldash/config"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
)

var Handler fiber.Handler

func init() {
	Handler = session.New(session.Config{
		Extractor:      extractors.FromCookie(config.Session.CookieName),
		CookieHTTPOnly: true,
		CookieSameSite: fiber.CookieSameSiteLaxMode,
		CookiePath:     "/",
		IdleTimeout:    Lifetime,
	})
}

func Remember(context fiber.Ctx, accountID uint) {
	held := session.FromContext(context)
	if held == nil {
		return
	}

	held.Set(AccountKey, accountID)
}

func Remembered(context fiber.Ctx) uint {
	held := session.FromContext(context)
	if held == nil {
		return 0
	}

	accountID, matched := held.Get(AccountKey).(uint)
	if !matched {
		return 0
	}

	return accountID
}

func Forget(context fiber.Ctx) {
	held := session.FromContext(context)
	if held == nil {
		return
	}

	held.Destroy()
}
