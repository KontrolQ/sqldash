package postgresconnections

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/secretstore"
	"sqldash/services/access"
	databaseservice "sqldash/services/databases"
	credentialpending "sqldash/services/postgresconnections/pending"
	primarycredentials "sqldash/services/postgresconnections/primary"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type credentialPayload struct {
	Generation                uint64
	Applied                   bool
	SecretReference           string
	PreviousSecret            string
	PreviousPasswordReference string
	PrimaryConnection         bool
}

func authorize(ctx context.Context, name string, scope models.TokenScope) (context.Context, *models.Database, *fiber.Error) {
	fresh, record, err := databaseservice.Authorize(ctx, name, permissions.PostgresCredentialsManage, true)
	if err != nil {
		return fresh, nil, err
	}
	actor := access.FromContext(fresh)
	if record.Kind() != models.EnginePostgres || !actor.Can(permissions.PostgresDataRead) || scope == models.ScopeReadWrite && !actor.Can(permissions.PostgresDataWrite) {
		return fresh, nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	if scope != models.ScopeReadOnly && scope != models.ScopeReadWrite {
		return fresh, nil, shortcuts.ServiceError(http.StatusBadRequest, Invalid)
	}
	return fresh, record, nil
}

func Create(ctx context.Context, name, label string, scope models.TokenScope, expiresAt *time.Time) (*IssuedPassword, *fiber.Error) {
	return nil, shortcuts.ServiceError(http.StatusBadRequest, SingleConnection)
}

func Rotate(ctx context.Context, name string, identifier uint) (*IssuedPassword, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, models.ScopeReadOnly)
	if err != nil {
		return nil, err
	}
	connection, findError := findConnection(fresh, record.ID, identifier)
	if findError != nil {
		return nil, findError
	}
	if connection.PrimaryConnection {
		if _, authorityError := primarycredentials.Authorize(fresh, database.DB.WithContext(fresh), record, access.AccountID(fresh)); authorityError != nil {
			return nil, shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
		}
	}
	fresh, _, err = authorize(fresh, name, connection.Scope)
	if err != nil {
		return nil, err
	}
	if connection.RevokedAt != nil || connection.ExpiresAt != nil && !connection.ExpiresAt.After(time.Now()) {
		return nil, shortcuts.ServiceError(http.StatusConflict, Invalid)
	}
	connection.AccountID = access.AccountID(fresh)
	return reservePassword(fresh, record, connection, "credential-rotate")
}

func findConnection(ctx context.Context, databaseID, identifier uint) (*models.PostgresConnection, *fiber.Error) {
	var connection models.PostgresConnection
	if err := database.DB.WithContext(ctx).Where("id = ? AND database_id = ?", identifier, databaseID).First(&connection).Error; err != nil {
		return nil, shortcuts.ServiceError(http.StatusNotFound, Missing)
	}
	return &connection, nil
}

func pendingReference() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func reservePassword(ctx context.Context, record *models.Database, connection *models.PostgresConnection, kind string) (*IssuedPassword, *fiber.Error) {
	var result *IssuedPassword
	err := mutateCredential(ctx, record, access.AccountID(ctx), func(fenced context.Context) *fiber.Error {
		var actionError *fiber.Error
		result, actionError = reservePasswordUnderFence(fenced, record, connection, kind)
		return actionError
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func reservePasswordUnderFence(ctx context.Context, record *models.Database, connection *models.PostgresConnection, kind string) (*IssuedPassword, *fiber.Error) {
	reference, err := pendingReference()
	if err != nil {
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, Refused)
	}
	namespace := PendingNamespace
	if connection.PrimaryConnection {
		namespace = primarycredentials.SecretNamespace
		reference = primarycredentials.Reference(record.ID, connection.Generation+1) + "-" + reference
	}
	oldReference := connection.PendingSecret
	connection.PendingSecret = reference
	operation := &models.DatabaseOperation{DatabaseID: record.ID, AccountID: access.AccountID(ctx), Engine: models.EnginePostgres, Kind: kind, State: models.DatabasePending}
	err = database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		var current models.Database
		if err := transaction.First(&current, record.ID).Error; err != nil {
			return err
		}
		if !current.Available() || current.Kind() != models.EnginePostgres {
			return errors.New(Refused)
		}
		if connection.ID == 0 {
			connection.Username = "pending_" + reference
			if err := transaction.Create(connection).Error; err != nil {
				return err
			}
			username, err := control.ApplicationUsername(record, connection)
			if err != nil {
				return err
			}
			connection.Username = username
			if err := transaction.Model(connection).Update("username", username).Error; err != nil {
				return err
			}
		} else {
			var active int64
			if err := transaction.Model(&models.DatabaseOperation{}).Where("connection_id = ? AND state = ? AND lease_until > ?", connection.ID, models.DatabaseApplying, time.Now()).Count(&active).Error; err != nil {
				return err
			}
			if active != 0 {
				return errors.New(Refused)
			}
			result := transaction.Model(&models.PostgresConnection{}).Where("id = ? AND database_id = ? AND generation = ? AND revoked_at IS NULL", connection.ID, record.ID, connection.Generation).Updates(map[string]any{"generation": gorm.Expr("generation + 1"), "state": models.DatabasePending, "pending_secret": reference, "account_id": connection.AccountID})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New(Refused)
			}
			connection.Generation++
		}
		if err := transaction.Model(&models.DatabaseOperation{}).Where("connection_id = ? AND state <> ?", connection.ID, models.DatabaseReady).Updates(map[string]any{"state": "cancelled", "lease_until": nil}).Error; err != nil {
			return err
		}
		operation.ConnectionID = connection.ID
		payload, err := json.Marshal(credentialPayload{Generation: connection.Generation, SecretReference: reference, PreviousSecret: oldReference, PrimaryConnection: connection.PrimaryConnection, PreviousPasswordReference: connection.PasswordReference})
		if err != nil {
			return err
		}
		operation.Payload = string(payload)
		if err := transaction.Model(record).Update("policy_version", gorm.Expr("policy_version + 1")).Error; err != nil {
			return err
		}
		return transaction.Create(operation).Error
	})
	if err != nil {
		return nil, shortcuts.ServiceError(http.StatusConflict, Refused)
	}
	connection.State = models.DatabasePending
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return nil, shortcuts.ServiceError(http.StatusServiceUnavailable, Refused)
	}
	password, err := store.GetOrCreate(namespace, reference, credentialpending.GeneratePassword)
	if err != nil {
		return nil, shortcuts.ServiceError(http.StatusServiceUnavailable, Refused)
	}
	defer clear(password)
	if err := executeUnderFence(ctx, record, connection, operation); err != nil {
		return nil, err
	}
	result := issuedPassword(record, connection, string(password))
	return result, nil
}

func Revoke(ctx context.Context, name string, identifier uint) *fiber.Error {
	fresh, record, err := authorize(ctx, name, models.ScopeReadOnly)
	if err != nil {
		return err
	}
	connection, findError := findConnection(fresh, record.ID, identifier)
	if findError != nil {
		return findError
	}
	if connection.PrimaryConnection {
		return shortcuts.ServiceError(http.StatusConflict, PrimaryCannotBeRevoked)
	}
	return mutateCredential(fresh, record, access.AccountID(fresh), func(fenced context.Context) *fiber.Error {
		return revokeUnderFence(fenced, record, connection)
	})
}

func revokeUnderFence(fresh context.Context, record *models.Database, connection *models.PostgresConnection) *fiber.Error {
	now := time.Now()
	operation := &models.DatabaseOperation{DatabaseID: record.ID, ConnectionID: connection.ID, AccountID: access.AccountID(fresh), Engine: models.EnginePostgres, Kind: "credential-revoke", State: models.DatabasePending}
	var deferNative bool
	reserveError := database.DB.WithContext(fresh).Transaction(func(transaction *gorm.DB) error {
		var active []models.DatabaseOperation
		if err := transaction.Where("connection_id = ? AND state IN ? AND lease_until > ?", connection.ID, []string{models.DatabaseApplying, models.DatabasePending}, now).Order("lease_until desc").Limit(1).Find(&active).Error; err != nil {
			return err
		}
		if len(active) != 0 {
			deferNative = true
			operation.LeaseUntil = active[0].LeaseUntil
		}
		result := transaction.Model(connection).Where("generation = ?", connection.Generation).Updates(map[string]any{"state": "revoking", "revoked_at": now, "generation": gorm.Expr("generation + 1")})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New(Refused)
		}
		connection.Generation++
		encoded, err := json.Marshal(credentialPayload{Generation: connection.Generation})
		if err != nil {
			return err
		}
		operation.Payload = string(encoded)
		if err := transaction.Model(&models.DatabaseOperation{}).Where("connection_id = ? AND state <> ?", connection.ID, models.DatabaseReady).Updates(map[string]any{"state": "cancelled", "lease_until": nil}).Error; err != nil {
			return err
		}
		if err := transaction.Model(record).Update("policy_version", gorm.Expr("policy_version + 1")).Error; err != nil {
			return err
		}
		return transaction.Create(operation).Error
	})
	if reserveError != nil {
		return shortcuts.ServiceError(http.StatusConflict, Refused)
	}
	connection.State = "revoking"
	connection.RevokedAt = &now
	if deferNative {
		return shortcuts.ServiceError(http.StatusConflict, Refused)
	}
	return executeUnderFence(fresh, record, connection, operation)
}

func execute(parent context.Context, record *models.Database, connection *models.PostgresConnection, operation *models.DatabaseOperation) *fiber.Error {
	return mutateCredential(parent, record, operation.AccountID, func(fenced context.Context) *fiber.Error {
		return executeUnderFence(fenced, record, connection, operation)
	})
}

func executeUnderFence(parent context.Context, record *models.Database, connection *models.PostgresConnection, operation *models.DatabaseOperation) *fiber.Error {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	var current models.DatabaseOperation
	if err := database.DB.WithContext(ctx).Where("id = ? AND database_id = ? AND connection_id = ? AND kind IN ?", operation.ID, record.ID, connection.ID, []string{"credential-create", "credential-rotate", "credential-revoke"}).First(&current).Error; err != nil {
		return shortcuts.ServiceError(http.StatusConflict, Refused)
	}
	*operation = current
	lease := time.Now().Add(3 * time.Minute)
	claimed := database.DB.WithContext(ctx).Model(operation).Where("state IN ? AND (lease_until IS NULL OR lease_until < ?)", []string{models.DatabasePending, models.DatabaseApplying, models.DatabaseFailed}, time.Now()).Updates(map[string]any{"state": models.DatabaseApplying, "lease_until": lease, "attempts": gorm.Expr("attempts + 1")})
	if claimed.Error != nil || claimed.RowsAffected != 1 {
		return shortcuts.ServiceError(http.StatusConflict, Refused)
	}
	operation.LeaseUntil = &lease
	operation.Attempts++
	if err := applyCredential(ctx, record, connection, operation); err != nil {
		cleanup, cleanupCancel := context.WithTimeout(context.WithoutCancel(parent), 10*time.Second)
		defer cleanupCancel()
		retryAt := time.Now().Add(databaseservice.OperationRetryDelay(operation.Attempts))
		database.DB.WithContext(cleanup).Model(operation).Where("state = ? AND lease_until = ?", models.DatabaseApplying, lease).Updates(map[string]any{"state": models.DatabaseFailed, "lease_until": retryAt, "failure": Refused})
		database.DB.WithContext(cleanup).Model(connection).Where("generation = ? AND state <> ?", connection.Generation, "revoked").Update("state", models.DatabaseFailed)
		return shortcuts.ServiceError(http.StatusBadGateway, Refused)
	}
	return nil
}

func applyCredential(ctx context.Context, record *models.Database, connection *models.PostgresConnection, operation *models.DatabaseOperation) error {
	var payload credentialPayload
	if err := json.Unmarshal([]byte(operation.Payload), &payload); err != nil {
		return err
	}
	if payload.Generation != connection.Generation {
		return errors.New(Refused)
	}
	fresh, currentConnection, authorityError := credentialAuthorityWith(ctx, database.DB.WithContext(ctx), record, operation, payload.Generation)
	if authorityError != nil {
		return authorityError
	}
	*connection = *currentConnection
	if payload.PrimaryConnection != connection.PrimaryConnection {
		return errors.New(Refused)
	}
	namespace := PendingNamespace
	if connection.PrimaryConnection {
		namespace = primarycredentials.SecretNamespace
		if !primarycredentials.ValidReference(record, connection, connection.PendingSecret) {
			return errors.New(Refused)
		}
	}
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return errors.New(Refused)
	}
	if operation.Kind == "credential-revoke" {
		if err := control.RevokeLogin(fresh, record, connection); err != nil {
			return err
		}
	} else {
		if connection.AccountID != operation.AccountID || connection.RevokedAt != nil || connection.ExpiresAt != nil && !connection.ExpiresAt.After(time.Now()) {
			return errors.New(AuthorityRefused)
		}
		password := []byte{}
		if payload.SecretReference != "" && payload.SecretReference != connection.PendingSecret {
			return errors.New(Refused)
		}
		if !payload.Applied {
			if payload.SecretReference != "" {
				password, storeError = store.GetOrCreate(namespace, payload.SecretReference, credentialpending.GeneratePassword)
			} else {
				password, storeError = store.Get(namespace, connection.PendingSecret)
			}
			if storeError != nil {
				return errors.New(Refused)
			}
			defer clear(password)
		}
		var applyError error
		if operation.Kind == "credential-rotate" && !payload.Applied {
			applyError = control.RotateLogin(fresh, record, connection, string(password))
		} else {
			applyError = control.ReconcileLogin(fresh, record, connection, string(password))
		}
		if applyError != nil {
			return applyError
		}
		if _, _, err := authorize(fresh, record.Name, connection.Scope); err != nil {
			return err
		}
		payload.Applied = true
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		operation.Payload = string(encoded)
		checkpoint := database.DB.WithContext(ctx).Model(operation).Where("state = ? AND lease_until = ?", models.DatabaseApplying, operation.LeaseUntil).Update("payload", operation.Payload)
		if checkpoint.Error != nil {
			return checkpoint.Error
		}
		if checkpoint.RowsAffected != 1 {
			return errors.New(Refused)
		}
	}
	if connection.PrimaryConnection {
		for _, reference := range []string{payload.PreviousPasswordReference, payload.PreviousSecret} {
			if reference != "" && reference != connection.PendingSecret {
				if err := primarycredentials.DeleteReference(reference); err != nil {
					return err
				}
			}
		}
	} else if err := credentialpending.Clear(ctx, record.ID, connection.ID, connection.PendingSecret, payload.PreviousSecret); err != nil {
		return err
	}
	state := models.DatabaseReady
	if operation.Kind == "credential-revoke" {
		state = "revoked"
	}
	return database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if _, _, err := credentialAuthorityWith(ctx, transaction, record, operation, payload.Generation); err != nil {
			return err
		}
		properties := map[string]any{"state": state, "pending_secret": ""}
		if connection.PrimaryConnection {
			properties["password_reference"] = connection.PendingSecret
		}
		result := transaction.Model(connection).Where("generation = ?", payload.Generation).Updates(properties)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New(Refused)
		}
		completion := transaction.Model(operation).Where("state = ? AND lease_until = ?", models.DatabaseApplying, operation.LeaseUntil).Updates(map[string]any{"state": models.DatabaseReady, "lease_until": nil, "failure": ""})
		if completion.Error != nil {
			return completion.Error
		}
		if completion.RowsAffected != 1 {
			return errors.New(Refused)
		}
		return nil
	})
}

func Reconcile(ctx context.Context) error {
	var operations []models.DatabaseOperation
	if err := database.DB.WithContext(ctx).Where("kind IN ? AND state IN ? AND (lease_until IS NULL OR lease_until < ?)", []string{"credential-create", "credential-rotate", "credential-revoke"}, []string{models.DatabasePending, models.DatabaseApplying, models.DatabaseFailed}, time.Now()).Order("attempts asc, id asc").Limit(25).Find(&operations).Error; err != nil {
		return err
	}
	var failures []error
	for index := range operations {
		operation := &operations[index]
		var record models.Database
		var connection models.PostgresConnection
		if err := database.DB.WithContext(ctx).First(&record, operation.DatabaseID).Error; err != nil {
			failures = append(failures, err)
			continue
		}
		if err := database.DB.WithContext(ctx).Where("id = ? AND database_id = ?", operation.ConnectionID, record.ID).First(&connection).Error; err != nil {
			failures = append(failures, err)
			continue
		}
		if err := execute(ctx, &record, &connection, operation); err != nil {
			failures = append(failures, fmt.Errorf("connection %d: %s", connection.ID, Refused))
		}
	}
	return errors.Join(failures...)
}
