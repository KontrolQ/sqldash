package tokens

import (
	"net/http"
	"net/url"

	"sqldash/services/audit"
	databaseservice "sqldash/services/databases"
	service "sqldash/services/tokens"
	"sqldash/sessions"
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
		sessions.Complain(context, mintError.Message)
		return backTo(context, name)
	}

	audit.Note(context, name, audit.TokenMinted, asked.Label+MintedSuffix+asked.Scope)

	data, dataError := databaseservice.GetShowData(context.Context(), name, secret)
	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, ShowTemplate, data)
}

func Revoke(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	if revokeError := service.Revoke(context.Params(IdentifierParameter)); revokeError != nil {
		sessions.Complain(context, revokeError.Message)
	} else {
		sessions.Report(context, TokenRevoked)
		audit.Note(context, name, audit.TokenRevoked, "")
	}

	return backTo(context, name)
}

func backTo(context fiber.Ctx, name string) error {
	return shortcuts.RedirectToPath(context, ShowPath+url.PathEscape(name))
}
