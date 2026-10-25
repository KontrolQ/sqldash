package databases

import (
	"sqldash/services/audit"
	service "sqldash/services/databases"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func ExportRedis(ctx fiber.Ctx) error {
	name := ctx.Params(NameParameter)
	held, err := service.RedisExport(ctx.Context(), name)
	if err != nil {
		return shortcuts.ServiceError(fiber.StatusBadGateway, service.ExportRefused)
	}
	ctx.Set(DispositionHeader, `attachment; filename="`+name+`.rdb"`)
	ctx.Set(ContentTypeHeader, "application/octet-stream")
	ctx.Set("Cache-Control", "no-store")
	ctx.Set("Referrer-Policy", "no-referrer")
	audit.Note(ctx, name, audit.DatabaseExported, "Redis snapshot generated")
	return ctx.SendStream(held)
}
