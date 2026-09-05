package databases

import (
	"net/http"
	"net/url"

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
	}

	return backTo(context, name)
}

func backTo(context fiber.Ctx, name string) error {
	return shortcuts.RedirectToPath(context, ShowPath+url.PathEscape(name))
}
