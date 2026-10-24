package primary

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/secretstore"
	"sqldash/services/access"

	"gorm.io/gorm"
)

const SecretNamespace = "postgres-primary"

var Unavailable = errors.New("The database credentials are unavailable. Rotate the password to create a new managed password.")
var Refused = errors.New("You do not have permission to manage this database password.")

func Reference(databaseID uint, generation uint64) string {
	return fmt.Sprintf("%d-%d", databaseID, generation)
}

func ValidReference(record *models.Database, connection *models.PostgresConnection, reference string) bool {
	if reference == "" || connection == nil || connection.DatabaseID != record.ID {
		return false
	}
	prefix := Reference(record.ID, connection.Generation)
	return reference == prefix || strings.HasPrefix(reference, prefix+"-")
}

func Authorize(ctx context.Context, store *gorm.DB, record *models.Database, accountID uint) (*access.AccountAccess, error) {
	actor, err := access.LoadWith(store, accountID)
	if err != nil || actor == nil || !actor.Account.Active || !actor.Can(permissions.PostgresCredentialsManage) || !actor.Can(permissions.PostgresDataRead) || !actor.Can(permissions.PostgresDataWrite) || !actor.Can(permissions.PostgresObjectsManage) {
		return nil, Refused
	}
	var current models.Database
	if err := access.Scope(store.Model(&models.Database{}), actor).Where("id = ? AND name = ?", record.ID, record.Name).First(&current).Error; err != nil {
		return nil, Refused
	}
	if current.Kind() != models.EnginePostgres || current.State == models.DatabaseDeleting {
		return nil, Refused
	}
	return actor, nil
}

func Connection(ctx context.Context, record *models.Database) (*models.PostgresConnection, error) {
	var connection models.PostgresConnection
	err := database.DB.WithContext(ctx).Where("database_id = ? AND primary_connection = ?", record.ID, true).First(&connection).Error
	if err != nil {
		return nil, err
	}
	if connection.Username != record.Name || connection.Scope != models.ScopeReadWrite {
		return nil, Unavailable
	}
	return &connection, nil
}

func Password(record *models.Database, connection *models.PostgresConnection) (string, error) {
	if connection == nil || connection.DatabaseID != record.ID || !connection.PrimaryConnection || !connection.Usable(time.Now()) || connection.Username != record.Name || !ValidReference(record, connection, connection.PasswordReference) {
		return "", Unavailable
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return "", Unavailable
	}
	value, err := store.Get(SecretNamespace, connection.PasswordReference)
	if err != nil {
		return "", Unavailable
	}
	defer clear(value)
	return string(value), nil
}

func Ensure(ctx context.Context, record *models.Database, accountID uint) error {
	return Mutate(ctx, record, accountID, func(fenced context.Context) error {
		return ensure(fenced, record, accountID)
	})
}

func ensure(ctx context.Context, record *models.Database, accountID uint) error {
	if _, err := Authorize(ctx, database.DB.WithContext(ctx), record, accountID); err != nil {
		return err
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return Unavailable
	}
	connection, findError := Connection(ctx, record)
	if findError != nil && !errors.Is(findError, gorm.ErrRecordNotFound) {
		return findError
	}
	var password []byte
	var reference string
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		reference = Reference(record.ID, 1)
		password, err = store.GetOrCreate(SecretNamespace, reference, func() ([]byte, error) {
			generated, err := control.GeneratePassword()
			return []byte(generated), err
		})
		if err != nil {
			return Unavailable
		}
		connection = &models.PostgresConnection{DatabaseID: record.ID, AccountID: accountID, Label: "Database connection", Username: record.Name, Scope: models.ScopeReadWrite, State: models.DatabasePending, Generation: 1, PrimaryConnection: true, PendingSecret: reference}
		if err := database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
			if _, err := Authorize(ctx, transaction, record, accountID); err != nil {
				return err
			}
			return transaction.Create(connection).Error
		}); err != nil {
			clear(password)
			return Unavailable
		}
	} else {
		var reserved int64
		if err := database.DB.WithContext(ctx).Model(&models.DatabaseOperation{}).Where("connection_id = ? AND kind IN ? AND state IN ?", connection.ID, []string{"credential-create", "credential-rotate"}, []string{models.DatabasePending, models.DatabaseApplying, models.DatabaseFailed}).Count(&reserved).Error; err != nil {
			return err
		}
		if reserved != 0 {
			return Unavailable
		}
		reference = connection.PasswordReference
		if connection.State != models.DatabaseReady {
			reference = connection.PendingSecret
		}
		if !ValidReference(record, connection, reference) {
			return Unavailable
		}
		password, err = store.Get(SecretNamespace, reference)
		if err != nil {
			return Unavailable
		}
	}
	defer clear(password)
	var current models.PostgresConnection
	if err := database.DB.WithContext(ctx).Where("id = ? AND database_id = ? AND generation = ?", connection.ID, record.ID, connection.Generation).First(&current).Error; err != nil {
		return Unavailable
	}
	if current.AccountID != accountID || current.RevokedAt != nil || current.ExpiresAt != nil || current.PendingSecret != connection.PendingSecret || current.PasswordReference != connection.PasswordReference {
		return Unavailable
	}
	if _, err := Authorize(ctx, database.DB.WithContext(ctx), record, accountID); err != nil {
		return err
	}
	if err := control.ReconcileLogin(ctx, record, &current, string(password)); err != nil {
		return err
	}
	return database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if _, err := Authorize(ctx, transaction, record, accountID); err != nil {
			return err
		}
		saved := transaction.Model(connection).Where("generation = ? AND revoked_at IS NULL", connection.Generation).Updates(map[string]any{"state": models.DatabaseReady, "password_reference": reference, "pending_secret": ""})
		if saved.Error != nil {
			return saved.Error
		}
		if saved.RowsAffected != 1 {
			return Unavailable
		}
		return nil
	})
}

func Reconcile(ctx context.Context, record *models.Database) error {
	connection, findError := Connection(ctx, record)
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil
	}
	if findError != nil {
		return findError
	}
	return Mutate(ctx, record, connection.AccountID, func(fenced context.Context) error {
		current, currentError := Connection(fenced, record)
		if currentError != nil {
			return currentError
		}
		if current.RevokedAt != nil || current.ExpiresAt != nil && !time.Now().Before(*current.ExpiresAt) {
			return control.RevokeLogin(fenced, record, current)
		}
		if current.State != models.DatabaseReady {
			return nil
		}
		authorized, authorityError := reconcileAuthority(fenced, record, current)
		if !authorized || authorityError != nil {
			return authorityError
		}
		if reconcileError := control.ReconcileLogin(fenced, record, current, ""); reconcileError != nil {
			return reconcileError
		}
		_, authorityError = reconcileAuthority(fenced, record, current)
		return authorityError
	})
}

func reconcileAuthority(ctx context.Context, record *models.Database, connection *models.PostgresConnection) (bool, error) {
	if _, authorityError := Authorize(ctx, database.DB.WithContext(ctx), record, connection.AccountID); authorityError != nil {
		if !errors.Is(authorityError, Refused) {
			return false, authorityError
		}
		return false, control.SuspendLogin(ctx, record, connection)
	}
	return true, nil
}

func DeleteReference(reference string) error {
	if reference == "" {
		return nil
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return Unavailable
	}
	return store.Delete(SecretNamespace, reference)
}
