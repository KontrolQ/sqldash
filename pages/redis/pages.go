package redis

import (
	databasePages "sqldash/pages/databases"
	service "sqldash/services/redis"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func render(context fiber.Ctx, section string) error {
	data, err := service.Page(context.Context(), context.Params("name"), section)
	if err != nil {
		return err
	}
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "same-origin")
	context.Set("X-Content-Type-Options", "nosniff")
	for action, allowed := range data.Actions {
		context.Locals(action, allowed)
	}
	context.Locals("DatabaseEngine", data.Engine)
	context.Locals("IsRedis", true)
	meta.SetPageTitle(context, data.Title)
	meta.SetCrumbs(context, meta.Crumb{Label: "Databases", URL: "/"}, meta.Crumb{Label: data.Name, URL: data.BasePath}, meta.Crumb{Label: data.Title})
	meta.SetSection(context, data.Name, section)
	return shortcuts.Render(context, "redis/"+section, data)
}

func Overview(context fiber.Ctx) error    { return databasePages.Show(context) }
func Data(context fiber.Ctx) error        { return render(context, "data") }
func Console(context fiber.Ctx) error     { return render(context, "console") }
func Insights(context fiber.Ctx) error    { return render(context, "insights") }
func Commands(context fiber.Ctx) error    { return render(context, "queries") }
func SlowLog(context fiber.Ctx) error     { return render(context, "slowlog") }
func Monitor(context fiber.Ctx) error     { return render(context, "monitor") }
func PubSub(context fiber.Ctx) error      { return render(context, "pubsub") }
func Connections(context fiber.Ctx) error { return render(context, "connections") }
