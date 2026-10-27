package databases

import (
	"context"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"sqldash/database"
	"sqldash/permissions"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveDisplayName(ctx context.Context, name string, displayName string) *fiber.Error {
	fresh, record, authorityError := Authorize(ctx, name, permissions.DatabaseSettings, false)
	if authorityError != nil {
		return authorityError
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || !utf8.ValidString(displayName) || utf8.RuneCountInString(displayName) > MaximumDisplayNameLength || strings.ContainsFunc(displayName, unicode.IsControl) {
		return shortcuts.ServiceError(http.StatusBadRequest, DisplayNameInvalid)
	}
	result := database.DB.WithContext(fresh).Model(record).Where("name = ?", record.Name).Update("display_name", displayName)
	if result.Error != nil || result.RowsAffected != 1 {
		return shortcuts.ServiceError(http.StatusInternalServerError, SettingsRefused)
	}
	return nil
}
