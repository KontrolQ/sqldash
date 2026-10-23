package databases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sqldash/config"
	"strings"
	"time"

	"sqldash/database"
	"sqldash/engines"
	"sqldash/hosting"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgmetrics"
	"sqldash/proxy"
	ruleRepository "sqldash/repositories/accessrule"
	snippetRepository "sqldash/repositories/snippet"
	tokenRepository "sqldash/repositories/token"
	credentialpending "sqldash/services/postgresconnections/pending"
	"sqldash/telemetry"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func Delete(ctx context.Context, name string) *fiber.Error {
	fresh, record, err := Authorize(ctx, name, permissions.DatabaseDelete, false)
	if err != nil {
		return err
	}
	if record.Protected {
		return shortcuts.ServiceError(http.StatusConflict, DatabaseHeld)
	}
	encoded, _ := json.Marshal(operationPayload{})
	operation := &models.DatabaseOperation{DatabaseID: record.ID, AccountID: record.OwnerAccountID, Engine: record.Kind(), Kind: "delete", State: models.DatabasePending, Payload: string(encoded)}
	_, actor, err := FreshActor(fresh)
	if err != nil {
		return err
	}
	operation.AccountID = actor.Account.ID
	reserveError := database.DB.WithContext(fresh).Transaction(func(transaction *gorm.DB) error {
		var active int64
		if err := transaction.Model(&models.DatabaseOperation{}).Where("database_id = ? AND state IN ? AND lease_until > ?", record.ID, []string{models.DatabasePending, models.DatabaseApplying, remoteStagingState}, time.Now()).Count(&active).Error; err != nil {
			return err
		}
		if active != 0 {
			return errors.New(DatabaseNotReady)
		}
		if err := transaction.Model(&models.DatabaseOperation{}).Where("database_id = ? AND state <> ?", record.ID, models.DatabaseReady).Updates(map[string]any{"state": "cancelled", "lease_until": nil}).Error; err != nil {
			return err
		}
		result := transaction.Model(record).Where("protected = ?", false).Updates(map[string]any{"state": models.DatabaseDeleting, "policy_version": gorm.Expr("policy_version + 1")})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New(DatabaseHeld)
		}
		now := time.Now()
		if err := transaction.Model(&models.PostgresConnection{}).Where("database_id = ?", record.ID).Updates(map[string]any{"state": "revoked", "revoked_at": now, "generation": gorm.Expr("generation + 1")}).Error; err != nil {
			return err
		}
		return transaction.Create(operation).Error
	})
	if reserveError != nil {
		return shortcuts.ServiceError(http.StatusConflict, DeleteRefused)
	}
	record.State = models.DatabaseDeleting
	proxy.Forget()
	return executeOperation(fresh, record, operation)
}

func clearImports(ctx context.Context, record *models.Database) error {
	var operations []models.DatabaseOperation
	if err := database.DB.WithContext(ctx).Where("database_id = ? AND kind IN ?", record.ID, []string{"import", "remote-clone"}).Find(&operations).Error; err != nil {
		return err
	}
	directory := config.ImportsPath()
	if record.Kind() == models.EnginePostgres {
		directory = config.PostgresStagingPath()
	}
	for _, operation := range operations {
		var payload operationPayload
		if err := json.Unmarshal([]byte(operation.Payload), &payload); err != nil {
			return err
		}
		if err := clearRemoteSecret(&operation); err != nil {
			return err
		}
		if payload.Path == "" {
			continue
		}
		relative, err := filepath.Rel(directory, payload.Path)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return errors.New(DeleteRefused)
		}
		if err := os.Remove(payload.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func finishDelete(ctx context.Context, record *models.Database, operation *models.DatabaseOperation, backend engines.Backend) error {
	if record.Protected {
		return errors.New(DatabaseHeld)
	}
	var payload operationPayload
	if err := json.Unmarshal([]byte(operation.Payload), &payload); err != nil {
		return err
	}
	if payload.Phase != "cleanup" {
		if err := backend.Delete(ctx, record); err != nil {
			return err
		}
		payload.Phase = "cleanup"
		if err := savePhase(ctx, operation, payload); err != nil {
			return err
		}
	}
	if err := clearImports(ctx, record); err != nil {
		return err
	}
	for _, cleanup := range []func(string) error{tokenRepository.DeleteForDatabase, ruleRepository.DeleteForDatabase, snippetRepository.DeleteForDatabase, telemetry.DeleteForDatabase} {
		if err := cleanup(record.Name); err != nil {
			return err
		}
	}
	if record.Kind() == models.EnginePostgres {
		if err := credentialpending.Clear(ctx, record.ID, 0); err != nil {
			return err
		}
		if err := pgmetrics.DeleteForDatabase(record.ID); err != nil {
			return err
		}
	}
	if record.Kind() == models.EngineLibSQL {
		hosting.Withdraw(ctx, record.Name)
	}
	proxy.Forget()
	return database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if _, _, err := operationAuthorityWith(ctx, transaction, record, operation); err != nil {
			return err
		}
		if err := transaction.Unscoped().Where("database_id = ?", record.ID).Delete(&models.PostgresConnection{}).Error; err != nil {
			return err
		}
		if err := transaction.Unscoped().Where("database_id = ?", record.ID).Delete(&models.DatabaseShare{}).Error; err != nil {
			return err
		}
		completed := transaction.Model(operation).Where("state = ? AND lease_until = ?", models.DatabaseApplying, operation.LeaseUntil).Updates(map[string]any{"state": models.DatabaseReady, "failure": "", "lease_until": nil})
		if completed.Error != nil {
			return completed.Error
		}
		if completed.RowsAffected != 1 {
			return errors.New(DatabaseNotReady)
		}
		removed := transaction.Unscoped().Where("protected = ? AND state = ?", false, models.DatabaseDeleting).Delete(record)
		if removed.Error != nil {
			return removed.Error
		}
		if removed.RowsAffected != 1 {
			return errors.New(DatabaseHeld)
		}
		return nil
	})
}
