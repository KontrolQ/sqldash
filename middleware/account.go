package middleware

import (
	"strings"
	"time"

	store "sqldash/database"
	"sqldash/models"

	repository "sqldash/repositories/account"
	"sqldash/services/access"
	service "sqldash/services/auth"
	"sqldash/sessions"
	"sqldash/utils/logger"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func account(context fiber.Ctx) error {
	path := context.Path()

	if strings.HasPrefix(path, StaticPath) {
		return context.Next()
	}

	accountID := sessions.Remembered(context)

	if accountID != 0 {
		held, findError := repository.FindByID(accountID)
		if findError != nil {
			logger.Errorf(LogPrefix, AccountLookupFailedLog, findError)
		}

		if held != nil && held.Active && sessions.Version(context) == held.SessionVersion {
			granted, accessError := access.Load(held.ID)
			if accessError != nil {
				return accessError
			}
			context.Locals(access.ContextKey, granted)
			context.SetContext(access.WithContext(context.Context(), granted))
			context.Locals(meta.AccountKey, service.ToAccountView(held))
			context.Locals("CanManageUsers", granted.Can("users.manage"))
			context.Locals("CanManageRoles", granted.Can("roles.manage"))
			context.Locals("CanManageSMTP", granted.Can("settings.smtp"))
			context.Locals("IsSiteOwner", granted.Owner)
			for key, permission := range map[string]string{"CanViewDatabases": "databases.view", "CanCreateDatabases": "databases.create", "CanViewInsights": "insights.view", "CanViewAudit": "audit.view", "CanReadData": "data.read", "CanWriteData": "data.write", "CanRunSQL": "sql.execute", "CanManageTokens": "tokens.manage", "CanManageNetwork": "network.manage", "CanEditDatabaseSettings": "databases.settings", "CanDeleteDatabases": "databases.delete", "CanImportDatabases": "databases.import", "CanExportDatabases": "databases.export", "CanForkDatabases": "databases.fork"} {
				context.Locals(key, granted.Can(permission))
			}
			var requests []models.OwnershipTransfer
			if transferError := store.DB.Where("status = ? AND expires_at > ? AND (recipient_account_id = ? OR owner_account_id = ?)", "pending", time.Now(), held.ID, held.ID).Find(&requests).Error; transferError != nil {
				return transferError
			}
			context.Locals("OwnershipRequests", requests)
			context.Locals("CurrentAccountID", held.ID)
			return context.Next()
		}

		if forgetError := sessions.Forget(context); forgetError != nil {
			return forgetError
		}
	}

	if isOpen(path) {
		return context.Next()
	}

	needed, guardError := service.NeedsSetup()
	if guardError != nil {
		return guardError
	}

	if needed {
		return shortcuts.RedirectToPath(context, SetupPath)
	}

	remember(context)

	return shortcuts.RedirectToPath(context, LoginPath)
}

func remember(context fiber.Ctx) {
	if context.Method() != fiber.MethodGet {
		return
	}

	if !strings.Contains(context.Get(fiber.HeaderAccept), HTMLContent) {
		return
	}

	wanted := context.OriginalURL()

	if !strings.HasPrefix(wanted, "/") || strings.HasPrefix(wanted, "//") {
		return
	}

	if wanted == LoginPath || wanted == SetupPath {
		return
	}

	sessions.Intend(context, wanted)
}

func isOpen(path string) bool {
	for _, open := range OpenPaths {
		if path == open {
			return true
		}
	}

	return false
}
