package databases

import (
	"net/http"
	"net/url"

	service "sqldash/services/databases"
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
		return backTo(context, name, ProblemParameter, saveError.Message)
	}

	return backTo(context, name, DoneParameter, SettingsSaved)
}

func backTo(context fiber.Ctx, name string, parameter string, message string) error {
	return shortcuts.RedirectToPath(
		context,
		ShowPath+url.PathEscape(name)+"?"+parameter+"="+url.QueryEscape(message),
	)
}
