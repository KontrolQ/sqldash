package middleware

import (
	"strings"

	repository "sqldash/repositories/account"
	service "sqldash/services/auth"
	"sqldash/sessions"
	"sqldash/utils/logger"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func account(context fiber.Ctx) error {
	path := context.Path()

	if strings.HasPrefix(path, StaticPath) {
		return context.Next()
	}

	accountID := sessions.Remembered(context)

	if accountID != 0 {
		held, findError := repository.FindByID(accountID)
		if findError != nil {
			logger.Errorf(LogPrefix, AccountLookupFailedLog, findError)
		}

		if held != nil {
			context.Locals(meta.AccountKey, service.ToAccountView(held))
			return context.Next()
		}

		sessions.Forget(context)
	}

	if isOpen(path) {
		return context.Next()
	}

	needed, guardError := service.NeedsSetup()
	if guardError != nil {
		return guardError
	}

	if needed {
		return shortcuts.RedirectToPath(context, SetupPath)
	}

	remember(context)

	return shortcuts.RedirectToPath(context, LoginPath)
}

func remember(context fiber.Ctx) {
	if context.Method() != fiber.MethodGet {
		return
	}

	if !strings.Contains(context.Get(fiber.HeaderAccept), HTMLContent) {
		return
	}

	wanted := context.OriginalURL()

	if !strings.HasPrefix(wanted, "/") || strings.HasPrefix(wanted, "//") {
		return
	}

	if wanted == LoginPath || wanted == SetupPath {
		return
	}

	sessions.Intend(context, wanted)
}

func isOpen(path string) bool {
	for _, open := range OpenPaths {
		if path == open {
			return true
		}
	}

	return false
}
