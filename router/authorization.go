package router

import (
	"github.com/gofiber/fiber/v3"
	"net/url"
	"sqldash/engines"
	"sqldash/models"
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
	case "databases.settings", "databases.settings.postgres":
		return permissions.DatabaseSettings
	case "databases.delete":
		return permissions.DatabaseDelete
	case "databases.import":
		return permissions.DatabaseImport
	case "databases.import.remote":
		return permissions.PostgresDatabaseImport
	case "databases.export", "databases.export.file":
		return permissions.DatabaseExport
	case "databases.export.postgres.archive", "databases.export.postgres.sql":
		return permissions.PostgresDatabaseExport
	case "postgres.certificate":
		return permissions.PostgresDataRead
	case "databases.fork":
		return permissions.DatabaseFork
	case "databases.explore", "databases.structure":
		return permissions.DataRead
	case "databases.explore.insert", "databases.explore.cell", "databases.explore.row":
		return permissions.DataWrite
	case "databases.console", "databases.console.run", "databases.console.cancel", "databases.console.snippet.save", "databases.console.snippet.remove":
		return permissions.SQLExecute
	case "databases.tokens.mint", "databases.tokens.revoke":
		return permissions.TokenManage
	case "databases.rules.add", "databases.rules.remove":
		return permissions.NetworkManage
	case "databases.connections.create", "databases.connections.rotate", "databases.connections.revoke", "databases.connections.rotate.primary", "databases.connections.reveal":
		return permissions.PostgresCredentialsManage
	case "databases.structure.object":
		return permissions.PostgresObjectsManage
	case "databases.connections", "databases.activity":
		return permissions.PostgresActivityView
	case "databases.activity.cancel", "databases.activity.terminate":
		return permissions.PostgresActivityManage
	case "databases.extensions", "databases.extensions.manage":
		return permissions.PostgresExtensionsManage
	case "databases.certificate":
		return permissions.PostgresDatabaseView
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
		name := route.Name
		if route.Namespace != "" {
			name = route.Namespace + "." + name
		}
		credentialLink := name == "auth.join.accept" || name == "auth.recover.accept"
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
		held, _ := context.Locals(access.ContextKey).(*access.AccountAccess)
		needed := routeCapability(name)
		if strings.Contains(route.FullPath, "/databases/:name") {
			record, findError := access.Database(held, context.Params("name"))
			if findError != nil {
				return fiber.ErrNotFound
			}
			if guardError := guardDatabase(context, held, record, name, needed); guardError != nil {
				return guardError
			}
		} else if needed != "" {
			allowed := held.CanAny(needed)
			if needed == permissions.DatabaseCreate || needed == permissions.DatabaseImport {
				kind := context.FormValue("engine", models.EngineLibSQL)
				if kind != models.EngineLibSQL && kind != models.EnginePostgres {
					return fiber.ErrBadRequest
				}
				allowed = held.Can(permissions.ForEngine(needed, kind))
			}
			if name == "audit" {
				allowed = allowed || held.Can(permissions.PlatformAuditView)
			}
			if !allowed {
				if name == "databases" && held != nil && context.Method() == fiber.MethodGet {
					return context.Redirect().To("/settings")
				}
				return shortcuts.ServiceError(fiber.StatusForbidden, access.Refused.Error())
			}
		}
		return route.Handler(context)
	}
}

func guardDatabase(context fiber.Ctx, held *access.AccountAccess, record *models.Database, name string, needed string) error {
	sharing := name == "databases.share" || name == "databases.sharing"
	if sharing {
		if !held.CanShare(record) {
			return fiber.ErrForbidden
		}
	} else if needed == "" || !held.CanDatabase(record, needed) {
		return shortcuts.ServiceError(fiber.StatusForbidden, access.Refused.Error())
	}
	if record.Kind() == models.EnginePostgres && (strings.Contains(name, ".tokens.") || name == "databases.export.file") {
		return fiber.ErrNotFound
	}
	context.Locals("DatabaseEngine", record.Kind())
	context.Locals("DatabaseRecord", record)
	context.Locals("IsPostgres", record.Kind() == models.EnginePostgres)
	for key, allowed := range access.Actions(held, record) {
		context.Locals(key, allowed)
	}
	mode := engines.ReadOnly
	mapped := permissions.ForEngine(needed, record.Kind())
	if mapped == permissions.DataWrite || mapped == permissions.PostgresDataWrite {
		mode = engines.ReadWrite
	}
	if mapped == permissions.PostgresObjectsManage {
		mode = engines.Objects
	}
	if mapped == permissions.SQLExecute || mapped == permissions.PostgresSQLExecute {
		mode = engines.SQLMode(held.CanDatabase(record, permissions.DataWrite), held.CanDatabase(record, permissions.PostgresObjectsManage))
	}
	context.SetContext(engines.WithMode(engines.WithDatabase(context.Context(), record), mode))
	return nil
}
