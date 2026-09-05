package databases

import (
	"net/http"
	"net/url"
	"strings"

	"sqldash/services/audit"
	service "sqldash/services/databases"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveSettings(context fiber.Ctx) error {
	asked, parseError := meta.Body[SettingsRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	saveError := service.SaveSettings(
		context.Context(),
		name,
		asked.BlockReads == CheckedValue,
		asked.BlockWrites == CheckedValue,
		asked.BlockReason,
		asked.AllowAttach == CheckedValue,
		asked.Protected == CheckedValue,
	)

	if saveError != nil {
		sessions.Complain(context, saveError.Message)
	} else {
		sessions.Report(context, SettingsSaved)
		audit.Note(context, name, audit.SettingsSaved, describeSettings(asked))
	}

	return backTo(context, name)
}

func backTo(context fiber.Ctx, name string) error {
	return shortcuts.RedirectToPath(context, ShowPath+url.PathEscape(name))
}

func describeSettings(asked SettingsRequest) string {
	held := make([]string, 0, 4)

	if asked.BlockReads == CheckedValue {
		held = append(held, BlockedReads)
	}

	if asked.BlockWrites == CheckedValue {
		held = append(held, BlockedWrites)
	}

	if asked.AllowAttach == CheckedValue {
		held = append(held, AllowedAttach)
	}

	if asked.Protected == CheckedValue {
		held = append(held, Protected)
	}

	if len(held) == 0 {
		return NothingBlocked
	}

	return strings.Join(held, ", ")
}
