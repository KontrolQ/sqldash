package databases

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"sqldash/services/audit"
	"sqldash/services/redisconnections"
	"sqldash/sessions"
)

func RotateRedisPrimary(ctx fiber.Ctx) error {
	name := ctx.Params(NameParameter)
	ctx.Set("Cache-Control", "no-store")
	ctx.Set("Referrer-Policy", "no-referrer")
	issued, err := redisconnections.RotatePrimary(ctx.Context(), name)
	if err != nil {
		sessions.Complain(ctx, err.Message)
	} else {
		audit.Note(ctx, name, audit.RedisCredentialRotated, fmt.Sprintf("Credential %d", issued.CredentialID))
		issued.Password = ""
		issued.URL = ""
		sessions.Report(ctx, "Database password rotated.")
	}
	return backTo(ctx, name)
}
