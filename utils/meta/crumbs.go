package meta

import "github.com/gofiber/fiber/v3"

type Crumb struct {
	Label string
	URL   string
}

func SetCrumbs(context fiber.Ctx, crumbs ...Crumb) {
	context.Locals(CrumbsKey, crumbs)
}

func SetSection(context fiber.Ctx, database string, section string) {
	context.Locals(DatabaseKey, database)
	context.Locals(SectionKey, section)
}
