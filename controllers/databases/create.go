package databases

import (
	"net/http"
	"net/url"

	service "sqldash/services/databases"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Create(context fiber.Ctx) error {
	asked, parseError := meta.Body[CreateRequest](context)
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, FormUnreadable)
	}

	if createError := service.Create(context.Context(), asked.Name); createError != nil {
		return shortcuts.RedirectToPath(context, IndexPath+"?problem="+url.QueryEscape(createError.Message))
	}

	return shortcuts.Redirect(context, IndexRoute)
}

func Delete(context fiber.Ctx) error {
	if deleteError := service.Delete(context.Context(), context.Params(NameParameter)); deleteError != nil {
		return shortcuts.RedirectToPath(context, IndexPath+"?problem="+url.QueryEscape(deleteError.Message))
	}

	return shortcuts.Redirect(context, IndexRoute)
}
