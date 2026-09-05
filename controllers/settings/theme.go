package settings

import (
	"net/http"
	"slices"
	"time"

	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func ChooseTheme(context fiber.Ctx) error {
	asked, parseError := meta.Body[ThemeRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	wanted := asked.Theme
	if !slices.Contains(Themes, wanted) {
		wanted = Themes[0]
	}

	context.Cookie(&fiber.Cookie{
		Name:     ThemeCookie,
		Value:    wanted,
		Path:     "/",
		Expires:  time.Now().Add(ThemeLifetime),
		HTTPOnly: false,
		SameSite: fiber.CookieSameSiteLaxMode,
	})

	return shortcuts.RedirectToPath(context, IndexPath)
}
