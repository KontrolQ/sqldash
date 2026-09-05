package meta

import "github.com/gofiber/fiber/v3"

func Body[T any](context fiber.Ctx) (T, error) {
	var body T

	if bindError := context.Bind().Body(&body); bindError != nil {
		return body, bindError
	}

	return body, nil
}
