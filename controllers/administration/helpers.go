package administration

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"sqldash/services/access"
	"sqldash/services/audit"
	"sqldash/services/mailing"
	"sqldash/services/members"
	"sqldash/sessions"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"
	"strconv"
)

func actor(context fiber.Ctx) *access.AccountAccess {
	held, _ := context.Locals(access.ContextKey).(*access.AccountAccess)
	return held
}
func identifier(context fiber.Ctx) uint {
	value, _ := strconv.ParseUint(context.Params("identifier"), 10, 32)
	return uint(value)
}
func complete(context fiber.Ctx, action string, failure error, destination string) error {
	if failure != nil {
		message := OperationFailed
		for _, known := range []error{members.Refused, members.Invalid, members.LinkInvalid, members.PasswordWrong, members.DependenciesInvalid, members.Duplicate, mailing.SettingsInvalid, mailing.DeliveryFailed, mailing.SecretUnavailable} {
			if errors.Is(failure, known) {
				message = known.Error()
			}
		}
		logger.Errorf(LogPrefix, "Administration operation failed: %v", failure)
		sessions.Complain(context, message)
	} else {
		sessions.Report(context, Saved)
		audit.Note(context, context.Params("name"), action, "")
	}
	return shortcuts.RedirectToPath(context, destination)
}
