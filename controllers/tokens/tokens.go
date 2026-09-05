package tokens

import (
	"net/http"
	"net/url"

	databaseservice "sqldash/services/databases"
	service "sqldash/services/tokens"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Mint(context fiber.Ctx) error {
	asked, parseError := meta.Body[MintRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	secret, mintError := service.Mint(name, asked.Label, asked.Scope)
	if mintError != nil {
		return backTo(context, name, ProblemParameter, mintError.Message)
	}

	data, dataError := databaseservice.GetShowData(context.Context(), name, secret, "", "")
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, ShowTemplate, data)
}

func Revoke(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	if revokeError := service.Revoke(context.Params(IdentifierParameter)); revokeError != nil {
		return backTo(context, name, ProblemParameter, revokeError.Message)
	}

	return backTo(context, name, DoneParameter, TokenRevoked)
}

func backTo(context fiber.Ctx, name string, parameter string, value string) error {
	return shortcuts.RedirectToPath(
		context,
		ShowPath+url.PathEscape(name)+"?"+parameter+"="+url.QueryEscape(value),
	)
}
