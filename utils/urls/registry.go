package urls

import (
	"sync"

	"sqldash/utils/collections"

	"github.com/gofiber/fiber/v3"
)

type RegisteredRoute struct {
	Method    HTTPMethod
	Path      string
	Handler   fiber.Handler
	Namespace string
	Name      string
	FullPath  string
	Fallback  bool
}

type RouteRegistry struct {
	Mutex            sync.Mutex
	CurrentNamespace string
	Routes           collections.OrderedMap[string, RegisteredRoute]
}

var registry = &RouteRegistry{
	Routes: collections.OrderedMapOf[string, RegisteredRoute](),
}

func SetNamespace(namespace string) {
	registry.Mutex.Lock()
	defer registry.Mutex.Unlock()
	registry.CurrentNamespace = namespace
}
