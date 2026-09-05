package databases

import (
	"net/http"
	"strconv"

	service "sqldash/services/databases"
	"sqldash/sessions"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func AllowNetwork(context fiber.Ctx) error {
	asked, parseError := meta.Body[RuleRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	name := context.Params(NameParameter)

	if allowError := service.AllowNetwork(name, asked.Network, asked.Note); allowError != nil {
		sessions.Complain(context, allowError.Message)
	} else {
		sessions.Report(context, service.NetworkAllowed)
	}

	return backTo(context, name)
}

func RemoveNetwork(context fiber.Ctx) error {
	name := context.Params(NameParameter)

	identifier, parseError := strconv.ParseUint(context.Params(RuleParameter), 10, 64)
	if parseError != nil {
		sessions.Complain(context, service.NetworkMissing)
		return backTo(context, name)
	}

	if removeError := service.RemoveNetwork(name, uint(identifier)); removeError != nil {
		sessions.Complain(context, removeError.Message)
	} else {
		sessions.Report(context, service.NetworkRemoved)
	}

	return backTo(context, name)
}
