package postgresmanagement

import (
	"net/url"
	service "sqldash/services/postgresmanagement"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Activity(context fiber.Ctx) error {
	shown, err := service.GetActivity(context.Context(), context.Params("name"))
	if err != nil {
		return err
	}
	setPage(context, shown.Name, shown.Title, "activity")
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "same-origin")
	return shortcuts.Render(context, ActivityTemplate, shown)
}

func Extensions(context fiber.Ctx) error {
	shown, err := service.GetExtensions(context.Context(), context.Params("name"))
	if err != nil {
		return err
	}
	setPage(context, shown.Name, shown.Title, "extensions")
	return shortcuts.Render(context, ExtensionsTemplate, shown)
}

func setPage(context fiber.Ctx, name string, title string, section string) {
	meta.SetPageTitle(context, title)
	meta.SetCrumbs(context, meta.Crumb{Label: DatabasesLabel, URL: "/databases"}, meta.Crumb{Label: name, URL: "/databases/" + url.PathEscape(name)}, meta.Crumb{Label: title})
	meta.SetSection(context, name, section)
}
