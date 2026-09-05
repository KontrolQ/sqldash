package urls

import (
	"strings"

	"sqldash/utils/collections"

	"github.com/gofiber/fiber/v3"
)

type HTTPMethod string

const (
	Delete  HTTPMethod = "DELETE"
	Get     HTTPMethod = "GET"
	Head    HTTPMethod = "HEAD"
	Options HTTPMethod = "OPTIONS"
	Patch   HTTPMethod = "PATCH"
	Post    HTTPMethod = "POST"
	Put     HTTPMethod = "PUT"
)

func Path(method HTTPMethod, path string, handler fiber.Handler, name string) {
	register(method, path, handler, name, false)
}

func Fallback(method HTTPMethod, path string, handler fiber.Handler, name string) {
	register(method, path, handler, name, true)
}

func register(method HTTPMethod, path string, handler fiber.Handler, name string, fallback bool) {
	registry.Mutex.Lock()
	defer registry.Mutex.Unlock()

	namespace := registry.CurrentNamespace
	fullName := resolveFullName(namespace, name)
	fullPath := resolveFullPath(namespace, path)

	registry.Routes.Set(fullName, RegisteredRoute{
		Method:    method,
		Path:      path,
		Handler:   handler,
		Namespace: namespace,
		Name:      name,
		FullPath:  fullPath,
		Fallback:  fallback,
	})
}

func GetFullPath(routeName string) (string, bool) {
	registry.Mutex.Lock()
	defer registry.Mutex.Unlock()

	route, exists := registry.Routes.Get(routeName)
	if !exists {
		return "", false
	}

	return route.FullPath, true
}

func ResolvePath(routeName string, params collections.Record[string, string]) (string, bool) {
	registry.Mutex.Lock()
	defer registry.Mutex.Unlock()

	route, exists := registry.Routes.Get(routeName)
	if !exists {
		return "", false
	}

	resolved := route.FullPath
	for key, value := range params {
		resolved = strings.ReplaceAll(resolved, ":"+key, value)
	}

	return resolved, true
}

func resolveFullName(namespace string, name string) string {
	switch namespace {
	case "":
		return name
	default:
		return namespace + "." + name
	}
}

func resolveFullPath(namespace string, path string) string {
	switch namespace {
	case "":
		return ensureLeadingSlash(path)
	default:
		return "/" + namespace + ensureLeadingSlash(path)
	}
}

func bindPath(application *fiber.App, route RegisteredRoute) {
	switch route.Method {
	case Delete:
		application.Delete(route.FullPath, route.Handler)
	case Get:
		application.Get(route.FullPath, route.Handler)
	case Head:
		application.Head(route.FullPath, route.Handler)
	case Options:
		application.Options(route.FullPath, route.Handler)
	case Patch:
		application.Patch(route.FullPath, route.Handler)
	case Post:
		application.Post(route.FullPath, route.Handler)
	case Put:
		application.Put(route.FullPath, route.Handler)
	}
}

func ensureLeadingSlash(path string) string {
	switch strings.HasPrefix(path, "/") {
	case true:
		return path
	default:
		return "/" + path
	}
}
