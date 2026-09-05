package sessions

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func Complain(context fiber.Ctx, message string) {
	remember(context, ProblemKey, message)
}

func Report(context fiber.Ctx, message string) {
	remember(context, DoneKey, message)
}

func TakeMessages(context fiber.Ctx) (string, string) {
	return take(context, ProblemKey), take(context, DoneKey)
}

func remember(context fiber.Ctx, key string, message string) {
	held := session.FromContext(context)
	if held == nil || message == "" {
		return
	}

	held.Set(key, message)
}

func take(context fiber.Ctx, key string) string {
	held := session.FromContext(context)
	if held == nil {
		return ""
	}

	message, matched := held.Get(key).(string)
	if !matched || message == "" {
		return ""
	}

	held.Delete(key)

	return message
}

func Intend(context fiber.Ctx, path string) {
	remember(context, IntendedKey, path)
}

func Intended(context fiber.Ctx) string {
	return take(context, IntendedKey)
}
