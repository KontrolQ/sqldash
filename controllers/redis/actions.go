package redis

import (
	"bytes"
	"encoding/json"
	"io"
	"sqldash/services/audit"
	"strings"

	"sqldash/redisdata"
	service "sqldash/services/redis"

	"github.com/gofiber/fiber/v3"
)

type problem struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}
type success struct {
	OK bool `json:"ok"`
}

func decodeRequest(context fiber.Ctx, target any) *fiber.Error {
	if len(context.Body()) > service.RequestLimit() || !strings.HasPrefix(strings.ToLower(context.Get("Content-Type")), "application/json") {
		return fiber.NewError(400, service.InvalidRequest)
	}
	decoder := json.NewDecoder(bytes.NewReader(context.Body()))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return fiber.NewError(400, service.InvalidRequest)
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return fiber.NewError(400, service.InvalidRequest)
	}
	return nil
}

func respond(context fiber.Ctx, value any, err *fiber.Error) error {
	context.Set("Cache-Control", "no-store")
	context.Set("X-Content-Type-Options", "nosniff")
	if err != nil {
		return context.Status(err.Code).JSON(problem{Error: err.Message, Status: err.Code})
	}
	return context.JSON(value)
}

func Scan(context fiber.Ctx) error {
	var asked service.ScanRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	value, err := service.Scan(context.Context(), context.Params("name"), asked)
	return respond(context, value, err)
}
func Read(context fiber.Ctx) error {
	var asked service.ReadRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	value, err := service.Read(context.Context(), context.Params("name"), asked)
	return respond(context, value, err)
}
func Mutate(context fiber.Ctx) error {
	var asked redisdata.Mutation
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	value, err := service.Mutate(context.Context(), context.Params("name"), asked)
	return respond(context, value, err)
}
func MutateBatch(context fiber.Ctx) error {
	var asked service.BatchRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	value, err := service.MutateBatch(context.Context(), context.Params("name"), asked)
	return respond(context, value, err)
}
func Rename(context fiber.Ctx) error {
	var asked service.RenameRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	err := service.Rename(context.Context(), context.Params("name"), asked)
	return respond(context, success{OK: err == nil}, err)
}
func Delete(context fiber.Ctx) error {
	var asked service.DeleteRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	err := service.Delete(context.Context(), context.Params("name"), asked)
	return respond(context, success{OK: err == nil}, err)
}
func Console(context fiber.Ctx) error {
	var asked service.ConsoleRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	value, err := service.Execute(context.Context(), context.Params("name"), asked)
	return respond(context, value, err)
}
func Insights(context fiber.Ctx) error {
	value, err := service.Insights(context.Context(), context.Params("name"))
	return respond(context, value, err)
}
func Commands(context fiber.Ctx) error {
	value, err := service.Commands(context.Context(), context.Params("name"))
	return respond(context, value, err)
}
func SlowLog(context fiber.Ctx) error {
	value, err := service.SlowLog(context.Context(), context.Params("name"))
	return respond(context, value, err)
}
func ResetSlowLog(context fiber.Ctx) error {
	err := service.ResetSlowLog(context.Context(), context.Params("name"))
	return respond(context, success{OK: err == nil}, err)
}
func Clients(context fiber.Ctx) error {
	value, err := service.Clients(context.Context(), context.Params("name"))
	return respond(context, value, err)
}
func TerminateClient(context fiber.Ctx) error {
	var asked service.TerminateClientRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	err := service.TerminateClient(context.Context(), context.Params("name"), asked)
	if err == nil {
		audit.Note(context, context.Params("name"), audit.RedisClientTerminated, "Client "+asked.ID)
	}
	return respond(context, success{OK: err == nil}, err)
}
func Publish(context fiber.Ctx) error {
	var asked service.PublishRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	value, err := service.Publish(context.Context(), context.Params("name"), asked)
	return respond(context, value, err)
}
