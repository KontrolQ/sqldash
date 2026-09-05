package urls

import "github.com/gofiber/fiber/v3"

func Attach(application *fiber.App) {
	registry.Mutex.Lock()
	defer registry.Mutex.Unlock()

	for _, route := range registry.Routes.All() {
		if !route.Fallback {
			bindPath(application, route)
		}
	}

	for _, route := range registry.Routes.All() {
		if route.Fallback {
			bindPath(application, route)
		}
	}
}
