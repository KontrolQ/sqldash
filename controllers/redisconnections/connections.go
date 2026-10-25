package redisconnections

import (
	"fmt"
	"sqldash/services/audit"
	"strconv"
	"time"

	"sqldash/models"
	service "sqldash/services/redisconnections"
	"sqldash/utils/collections"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

type CreationRequest struct {
	Label     string `form:"label"`
	Scope     string `form:"scope"`
	ExpiresAt string `form:"expiresAt"`
	Subscribe string `form:"subscribe"`
	Publish   string `form:"publish"`
}

func secure(context fiber.Ctx) {
	context.Set("Cache-Control", "no-store")
	context.Set("Pragma", "no-cache")
	context.Set("Referrer-Policy", "no-referrer")
	context.Set("X-Content-Type-Options", "nosniff")
}

func boolean(value string) (bool, error) {
	if value == "" || value == "off" {
		return false, nil
	}
	if value == "on" {
		return true, nil
	}
	return strconv.ParseBool(value)
}

func reject(context fiber.Ctx, status int, message string) error {
	return context.Status(status).JSON(collections.Record[string, any]{"error": message})
}

func Create(context fiber.Ctx) error {
	secure(context)
	asked, err := meta.Body[CreationRequest](context)
	if err != nil {
		return reject(context, fiber.StatusBadRequest, invalidRequest)
	}
	subscribe, err := boolean(asked.Subscribe)
	if err != nil {
		return reject(context, fiber.StatusBadRequest, invalidRequest)
	}
	publish, err := boolean(asked.Publish)
	if err != nil {
		return reject(context, fiber.StatusBadRequest, invalidRequest)
	}
	var expiresAt *time.Time
	if asked.ExpiresAt != "" {
		parsed, err := time.Parse(expiryLayout, asked.ExpiresAt)
		if err != nil {
			return reject(context, fiber.StatusBadRequest, invalidRequest)
		}
		expiresAt = &parsed
	}
	issued, serviceError := service.Create(context.Context(), context.Params("name"), service.Creation{Label: asked.Label, Scope: models.TokenScope(asked.Scope), ExpiresAt: expiresAt, Subscribe: subscribe, Publish: publish})
	return complete(context, issued, serviceError, audit.RedisCredentialCreated)
}

func identifier(context fiber.Ctx) (uint, *fiber.Error) {
	parsed, err := strconv.ParseUint(context.Params("identifier"), 10, strconv.IntSize)
	if err != nil || parsed == 0 {
		return 0, fiber.NewError(fiber.StatusNotFound, service.Missing.Error())
	}
	return uint(parsed), nil
}

func Rotate(context fiber.Ctx) error {
	secure(context)
	identifier, err := identifier(context)
	if err != nil {
		return reject(context, err.Code, err.Message)
	}
	issued, serviceError := service.Rotate(context.Context(), context.Params("name"), identifier)
	return complete(context, issued, serviceError, audit.RedisCredentialRotated)
}
func RotatePrimary(context fiber.Ctx) error {
	secure(context)
	issued, serviceError := service.RotatePrimary(context.Context(), context.Params("name"))
	return complete(context, issued, serviceError, audit.RedisCredentialRotated)
}
func Reveal(context fiber.Ctx) error {
	secure(context)
	identifier, err := identifier(context)
	if err != nil {
		return reject(context, err.Code, err.Message)
	}
	issued, serviceError := service.Reveal(context.Context(), context.Params("name"), identifier)
	return complete(context, issued, serviceError, audit.RedisCredentialRevealed)
}
func Revoke(context fiber.Ctx) error {
	secure(context)
	identifier, err := identifier(context)
	if err != nil {
		return reject(context, err.Code, err.Message)
	}
	if err := service.Revoke(context.Context(), context.Params("name"), identifier); err != nil {
		return reject(context, err.Code, err.Message)
	}
	audit.Note(context, context.Params("name"), audit.RedisCredentialRevoked, fmt.Sprintf("Credential %d", identifier))
	return context.JSON(collections.Record[string, any]{"revoked": true})
}
func complete(context fiber.Ctx, issued *service.Issued, err *fiber.Error, action string) error {
	if err != nil {
		return reject(context, err.Code, err.Message)
	}
	audit.Note(context, context.Params("name"), action, fmt.Sprintf("Credential %d", issued.CredentialID))
	return context.JSON(issued)
}
