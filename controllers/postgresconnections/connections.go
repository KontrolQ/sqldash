package postgresconnections

import (
	"net/url"
	"strconv"
	"time"

	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/audit"
	databaseservice "sqldash/services/databases"
	service "sqldash/services/postgresconnections"
	"sqldash/sessions"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

type CreateRequest struct {
	Label  string `form:"label"`
	Scope  string `form:"scope"`
	Expiry string `form:"expiry"`
}

func Create(context fiber.Ctx) error {
	asked, parseError := meta.Body[CreateRequest](context)
	if parseError != nil {
		sessions.Complain(context, service.Invalid)
		return back(context)
	}
	var expiry *time.Time
	if asked.Expiry != "" {
		value, err := time.ParseInLocation(service.ExpiryLayout, asked.Expiry, time.UTC)
		if err != nil {
			sessions.Complain(context, service.Invalid)
			return back(context)
		}
		expiry = &value
	}
	issued, err := service.Create(context.Context(), context.Params("name"), asked.Label, models.TokenScope(asked.Scope), expiry)
	return complete(context, issued, err, service.CreatedAction)
}

func Rotate(context fiber.Ctx) error {
	identifier, err := strconv.ParseUint(context.Params("identifier"), 10, 64)
	if err != nil || identifier == 0 {
		sessions.Complain(context, service.Missing)
		return back(context)
	}
	issued, rotateError := service.Rotate(context.Context(), context.Params("name"), uint(identifier))
	return complete(context, issued, rotateError, service.RotatedAction)
}

func Revoke(context fiber.Ctx) error {
	identifier, err := strconv.ParseUint(context.Params("identifier"), 10, 64)
	if err != nil || identifier == 0 {
		sessions.Complain(context, service.Missing)
		return back(context)
	}
	_, record, recordError := databaseservice.Authorize(context.Context(), context.Params("name"), permissions.PostgresCredentialsManage, true)
	if recordError != nil {
		sessions.Complain(context, recordError.Message)
		return back(context)
	}
	if err := service.Revoke(context.Context(), context.Params("name"), uint(identifier)); err != nil {
		sessions.Complain(context, err.Message)
	} else {
		sessions.Report(context, service.Revoked)
		audit.RecordDatabase(audit.ActorOf(context), record, service.RevokedAction, "")
	}
	return back(context)
}

func complete(context fiber.Ctx, issued *service.IssuedPassword, err *fiber.Error, action string) error {
	context.Set("Cache-Control", "no-store")
	context.Set("Referrer-Policy", "no-referrer")
	if err != nil {
		sessions.Complain(context, err.Message)
		return back(context)
	}
	_, record, recordError := databaseservice.Authorize(context.Context(), context.Params("name"), permissions.PostgresCredentialsManage, true)
	if recordError != nil || record.ID != issued.DatabaseID {
		sessions.Complain(context, service.LostPassword)
		return back(context)
	}
	audit.RecordDatabase(audit.ActorOf(context), record, action, "")
	sessions.Report(context, service.Rotated)
	return back(context)
}

func back(context fiber.Ctx) error {
	return context.Redirect().Status(fiber.StatusSeeOther).To("/databases/" + url.PathEscape(context.Params("name")))
}
