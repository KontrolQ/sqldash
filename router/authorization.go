package router

import (
	"github.com/gofiber/fiber/v3"
	"net/url"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"
	"sqldash/utils/urls"
	"strings"
)

func init() { urls.Guard = guardRoute }

func routeCapability(name string) string {
	switch name {
	case "databases", "databases.show":
		return permissions.DatabaseView
	case "databases.create":
		return permissions.DatabaseCreate
	case "databases.settings":
		return permissions.DatabaseSettings
	case "databases.delete":
		return permissions.DatabaseDelete
	case "databases.import":
		return permissions.DatabaseImport
	case "databases.export", "databases.export.file":
		return permissions.DatabaseExport
	case "databases.fork":
		return permissions.DatabaseFork
	case "databases.explore", "databases.structure":
		return permissions.DataRead
	case "databases.explore.insert", "databases.explore.cell", "databases.explore.row":
		return permissions.DataWrite
	case "databases.console", "databases.console.run", "databases.console.snippet.save", "databases.console.snippet.remove":
		return permissions.SQLExecute
	case "databases.tokens.mint", "databases.tokens.revoke":
		return permissions.TokenManage
	case "databases.rules.add", "databases.rules.remove":
		return permissions.NetworkManage
	case "insights", "insights.queries", "databases.insights", "databases.queries":
		return permissions.InsightsView
	case "audit":
		return permissions.AuditView
	}
	if strings.HasPrefix(name, "users.") {
		return permissions.UsersManage
	}
	if strings.HasPrefix(name, "admin.roles") {
		return permissions.RolesManage
	}
	if strings.HasPrefix(name, "admin.smtp") {
		return permissions.SMTPManage
	}
	return ""
}

func guardRoute(route urls.RegisteredRoute) fiber.Handler {
	return func(context fiber.Ctx) error {
		credentialLink := route.Name == "auth.join.accept" || route.Name == "auth.recover.accept"
		if !credentialLink && context.Method() != fiber.MethodGet && context.Method() != fiber.MethodHead && context.Method() != fiber.MethodOptions {
			if context.Get("Sec-Fetch-Site") == "cross-site" {
				return fiber.ErrForbidden
			}
			if origin := context.Get("Origin"); origin != "" {
				parsed, parseError := url.Parse(origin)
				if parseError != nil || parsed.User != nil || !strings.EqualFold(parsed.Host, string(context.Request().Host())) {
					return fiber.ErrForbidden
				}
			}
		}
		needed := routeCapability(route.Name)
		if route.Namespace != "" {
			needed = routeCapability(route.Namespace + "." + route.Name)
		}
		held, _ := context.Locals(access.ContextKey).(*access.AccountAccess)
		if needed != "" && (held == nil || !held.Can(needed)) {
			if route.Name == "databases" && held != nil && context.Method() == fiber.MethodGet {
				return context.Redirect().To("/settings")
			}
			return shortcuts.ServiceError(fiber.StatusForbidden, access.Refused.Error())
		}
		if strings.Contains(route.FullPath, "/databases/:name") {
			record, findError := access.Database(held, context.Params("name"))
			if findError != nil {
				return fiber.ErrNotFound
			}
			context.Locals("CanShareDatabase", record.OwnerAccountID == held.Account.ID || held.Can(permissions.DatabaseShare))
			if (route.Name == "databases.share" || route.Name == "databases.sharing") && record.OwnerAccountID != held.Account.ID && !held.Can(permissions.DatabaseShare) {
				return shortcuts.ServiceError(fiber.StatusForbidden, access.Refused.Error())
			}
		}
		return route.Handler(context)
	}
}
