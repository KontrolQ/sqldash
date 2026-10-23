package databases

import (
	"context"
	"net/http"

	"sqldash/database"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SaveSettings(ctx context.Context, name string, blockReads, blockWrites bool, reason string, allowAttach, protected bool) *fiber.Error {
	fresh, record, err := Authorize(ctx, name, permissions.DatabaseSettings, true)
	if err != nil {
		return err
	}
	if record.Kind() != models.EngineLibSQL {
		return shortcuts.ServiceError(http.StatusBadRequest, EngineRefused)
	}
	backend, providerError := engines.For(record)
	if providerError != nil {
		return shortcuts.ServiceError(http.StatusServiceUnavailable, SettingsRefused)
	}
	settings, settingsError := backend.Configuration(fresh, record)
	if settingsError != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, SettingsUnavailable)
	}
	settings.BlockReads, settings.BlockWrites, settings.AllowAttach = blockReads, blockWrites, allowAttach
	settings.BlockReason = nil
	if reason != "" {
		settings.BlockReason = &reason
	}
	if err := backend.SetConfiguration(fresh, record, *settings); err != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, SettingsRefused)
	}
	return saveProtection(fresh, record, protected)
}

func SavePostgresSettings(ctx context.Context, name string, connectionLimit, statementTimeoutMilliseconds, idleTransactionTimeoutMilliseconds int, protected bool) *fiber.Error {
	fresh, record, err := Authorize(ctx, name, permissions.DatabaseSettings, true)
	if err != nil {
		return err
	}
	if record.Kind() != models.EnginePostgres {
		return shortcuts.ServiceError(http.StatusBadRequest, EngineRefused)
	}
	options := control.Settings{ConnectionLimit: connectionLimit, StatementTimeoutMilliseconds: statementTimeoutMilliseconds, IdleTransactionTimeoutMilliseconds: idleTransactionTimeoutMilliseconds}
	if err := control.SetSettings(fresh, record, options); err != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, SettingsRefused)
	}
	return saveProtection(fresh, record, protected)
}

func saveProtection(ctx context.Context, record *models.Database, protected bool) *fiber.Error {
	fresh, current, err := Authorize(ctx, record.Name, permissions.DatabaseSettings, true)
	if err != nil {
		return err
	}
	if current.ID != record.ID {
		return shortcuts.ServiceError(http.StatusConflict, DatabaseMissing)
	}
	result := database.DB.WithContext(fresh).Model(record).Where("state = ? OR state = ?", models.DatabaseReady, "").Updates(map[string]any{"protected": protected, "policy_version": gorm.Expr("policy_version + 1")})
	if result.Error != nil || result.RowsAffected != 1 {
		return shortcuts.ServiceError(http.StatusInternalServerError, SettingsRefused)
	}
	return nil
}
