package redisconnections

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

var (
	Invalid     = errors.New("Redis credential input is invalid")
	Refused     = errors.New("Redis credential authority is no longer available")
	Missing     = errors.New("Redis credential was not found")
	Conflict    = errors.New("Redis credential changed; refresh and try again")
	Unavailable = errors.New("Redis credential policy could not be confirmed")
	Consumed    = errors.New("This Redis password is no longer available; rotate it to receive a new password")
)

func failure(err error) *fiber.Error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, Invalid):
		return fiber.NewError(fiber.StatusBadRequest, Invalid.Error())
	case errors.Is(err, Refused):
		return fiber.NewError(fiber.StatusForbidden, Refused.Error())
	case errors.Is(err, Missing):
		return fiber.NewError(fiber.StatusNotFound, Missing.Error())
	case errors.Is(err, Conflict):
		return fiber.NewError(fiber.StatusConflict, Conflict.Error())
	case errors.Is(err, Consumed):
		return fiber.NewError(fiber.StatusConflict, Consumed.Error())
	default:
		return fiber.NewError(fiber.StatusServiceUnavailable, Unavailable.Error())
	}
}
