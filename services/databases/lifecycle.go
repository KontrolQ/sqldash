package databases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"sqldash/database"
	"sqldash/engines"
	"sqldash/hosting"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/pgmetrics"
	"sqldash/pgtransfer"
	"sqldash/proxy"
	"sqldash/services/access"
	primarycredentials "sqldash/services/postgresconnections/primary"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const operationTimeout = 15 * time.Minute

var nameReservationConflict = errors.New(NameTaken)

func OperationRetryDelay(attempts uint) time.Duration {
	if attempts > 6 {
		return 5 * time.Minute
	}
	delay := time.Duration(uint64(1)<<attempts) * 5 * time.Second
	return min(delay, 5*time.Minute)
}

type operationPayload struct {
	DisplayName string
	Path        string
	Moment      *time.Time
	Phase       string
	Remote      *remotePayload
}

func reserveDatabase(ctx context.Context, name, engine string, accountID uint, kind string, sourceID uint, payload operationPayload) (*models.Database, *models.DatabaseOperation, *fiber.Error) {
	if engine == models.EngineLibSQL {
		exists, err := sqld.NamespaceExists(ctx, name)
		if err != nil {
			return nil, nil, shortcuts.ServiceError(http.StatusBadGateway, CreateRefused)
		}
		if exists {
			return nil, nil, shortcuts.ServiceError(http.StatusConflict, NameTaken)
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, shortcuts.ServiceError(http.StatusInternalServerError, OperationFailed)
	}
	record := &models.Database{Name: name, DisplayName: payload.DisplayName, Engine: engine, State: models.DatabasePending, OwnerAccountID: accountID, PolicyVersion: 1}
	operation := &models.DatabaseOperation{AccountID: accountID, Engine: engine, Kind: kind, State: models.DatabasePending, SourceDatabaseID: sourceID, Payload: string(encoded)}
	if kind == "remote-clone" {
		until := time.Now().Add(3 * time.Minute)
		operation.State = remoteStagingState
		operation.LeaseUntil = &until
	}
	err = database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		reservation := transaction.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).Create(record)
		if reservation.Error != nil {
			return reservation.Error
		}
		if reservation.RowsAffected == 0 {
			return nameReservationConflict
		}
		operation.DatabaseID = record.ID
		if err := transaction.Create(operation).Error; err != nil {
			return err
		}
		if kind == "remote-clone" {
			if payload.Remote == nil {
				return errors.New(RemoteRefused)
			}
			payload.Remote.PasswordReference = strconv.FormatUint(uint64(operation.ID), 10)
			encoded, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			operation.Payload = string(encoded)
			return transaction.Model(operation).Update("payload", operation.Payload).Error
		}
		return nil
	})
	if errors.Is(err, nameReservationConflict) {
		return nil, nil, shortcuts.ServiceError(http.StatusConflict, NameTaken)
	}
	if err != nil {
		logger.Errorf(LogPrefix, CreateFailedLog, name, err)
		return nil, nil, shortcuts.ServiceError(http.StatusInternalServerError, CreateRefused)
	}
	return record, operation, nil
}

func claimOperation(ctx context.Context, record *models.Database, operation *models.DatabaseOperation) error {
	now := time.Now()
	lease := now.Add(operationTimeout + time.Minute)
	if err := database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if _, _, err := operationAuthorityWith(ctx, transaction, record, operation); err != nil {
			return err
		}
		result := transaction.Model(&models.DatabaseOperation{}).Where("id = ? AND database_id = ? AND state IN ? AND (lease_until IS NULL OR lease_until < ?)", operation.ID, record.ID, []string{models.DatabasePending, models.DatabaseApplying, models.DatabaseFailed, remoteStagingState}, now).Updates(map[string]any{"state": models.DatabaseApplying, "lease_until": lease, "attempts": gorm.Expr("attempts + 1"), "failure": ""})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New(DatabaseNotReady)
		}
		if operation.Kind != "delete" {
			target := transaction.Model(&models.Database{}).Where("id = ? AND state <> ?", record.ID, models.DatabaseDeleting).Update("state", models.DatabaseApplying)
			if target.Error != nil {
				return target.Error
			}
			if target.RowsAffected != 1 {
				return errors.New(DatabaseNotReady)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	operation.State = models.DatabaseApplying
	operation.Attempts++
	operation.LeaseUntil = &lease
	if operation.Kind != "delete" {
		record.State = models.DatabaseApplying
	}
	return nil
}

func operationAuthority(ctx context.Context, record *models.Database, operation *models.DatabaseOperation) (context.Context, *models.Database, error) {
	return operationAuthorityWith(ctx, database.DB.WithContext(ctx), record, operation)
}

func operationAuthorityWith(ctx context.Context, store *gorm.DB, record *models.Database, operation *models.DatabaseOperation) (context.Context, *models.Database, error) {
	actor, err := access.LoadWith(store, operation.AccountID)
	if err != nil || actor == nil || !actor.Account.Active {
		return ctx, nil, errors.New(AuthorityRefused)
	}
	fresh := access.WithContext(ctx, actor)
	var target models.Database
	findError := access.Scope(store.Model(&models.Database{}), actor).Where("id = ? AND name = ?", record.ID, record.Name).First(&target).Error
	if findError != nil || target.ID != record.ID || target.Kind() != operation.Engine {
		return fresh, nil, errors.New(AuthorityRefused)
	}
	permission := permissions.DatabaseCreate
	switch operation.Kind {
	case "import", "remote-clone":
		permission = permissions.DatabaseImport
	case "clone":
		permission = permissions.DatabaseFork
	case "delete":
		permission = permissions.DatabaseDelete
	}
	if !actor.CanDatabase(&target, permission) {
		return fresh, nil, errors.New(AuthorityRefused)
	}
	if operation.Kind != "delete" && !canProvision(actor, target.Kind()) {
		return fresh, nil, errors.New(AuthorityRefused)
	}
	if operation.Kind == "import" || operation.Kind == "clone" || operation.Kind == "remote-clone" {
		if !actor.CanDatabase(&target, permissions.DatabaseCreate) {
			return fresh, nil, errors.New(AuthorityRefused)
		}
	}
	if operation.SourceDatabaseID == 0 {
		return fresh, nil, nil
	}
	var source models.Database
	if findError := access.Scope(store.Model(&models.Database{}), actor).First(&source, operation.SourceDatabaseID).Error; findError != nil {
		return fresh, nil, errors.New(DatabaseMissing)
	}
	if !source.Available() || source.Kind() != target.Kind() || !actor.CanDatabase(&source, permissions.DatabaseFork) || !actor.CanDatabase(&source, permissions.DataRead) {
		return fresh, nil, errors.New(AuthorityRefused)
	}
	return fresh, &source, nil
}

func executeOperation(parent context.Context, record *models.Database, operation *models.DatabaseOperation) *fiber.Error {
	ctx, cancel := context.WithTimeout(parent, operationTimeout)
	defer cancel()
	if err := claimOperation(ctx, record, operation); err != nil {
		return shortcuts.ServiceError(http.StatusConflict, DatabaseNotReady)
	}
	claimedLease := operation.LeaseUntil
	if err := applyOperation(ctx, record, operation); err != nil {
		message := OperationFailed
		if errors.Is(err, pgtransfer.RemoteExtensionsRequired) || err.Error() == ExtensionsMustBeManaged {
			message = ExtensionsMustBeManaged
		}
		cleanup, cleanupCancel := context.WithTimeout(context.WithoutCancel(parent), 10*time.Second)
		defer cleanupCancel()
		retryAt := time.Now().Add(OperationRetryDelay(operation.Attempts))
		database.DB.WithContext(cleanup).Transaction(func(transaction *gorm.DB) error {
			state := models.DatabaseFailed
			if operation.Kind == "delete" {
				state = models.DatabaseDeleting
			}
			failed := transaction.Model(&models.DatabaseOperation{}).Where("id = ? AND state = ? AND lease_until = ?", operation.ID, models.DatabaseApplying, claimedLease).Updates(map[string]any{"state": models.DatabaseFailed, "failure": message, "lease_until": retryAt})
			if failed.Error != nil {
				return failed.Error
			}
			if failed.RowsAffected != 1 {
				return nil
			}
			target := transaction.Model(&models.Database{}).Where("id = ?", record.ID)
			if operation.Kind != "delete" {
				target = target.Where("state <> ?", models.DatabaseDeleting)
			}
			return target.Updates(map[string]any{"state": state, "policy_version": gorm.Expr("policy_version + 1")}).Error
		})
		proxy.Forget()
		return shortcuts.ServiceError(http.StatusBadGateway, message)
	}
	return nil
}

func savePhase(ctx context.Context, operation *models.DatabaseOperation, payload operationPayload) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	operation.Payload = string(encoded)
	result := database.DB.WithContext(ctx).Model(operation).Where("lease_until = ?", operation.LeaseUntil).Update("payload", operation.Payload)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New(DatabaseNotReady)
	}
	return nil
}

func applyOperation(ctx context.Context, record *models.Database, operation *models.DatabaseOperation) error {
	var payload operationPayload
	if err := json.Unmarshal([]byte(operation.Payload), &payload); err != nil {
		return err
	}
	fresh, source, err := operationAuthority(ctx, record, operation)
	if err != nil {
		return err
	}
	var origin pgtransfer.RemoteSource
	if operation.Kind == "remote-clone" && payload.Phase != "verify" {
		origin, err = remoteSource(fresh, record, operation, &payload)
		if err != nil {
			return err
		}
		defer func() { origin.Password = "" }()
	}
	backend, err := engines.For(record)
	if err != nil {
		return err
	}
	if record.Kind() == models.EngineRedis {
		return applyRedisOperation(ctx, record, operation, backend, payload)
	}
	if operation.Kind == "delete" {
		return finishDelete(fresh, record, operation, backend)
	}
	if !record.Available() && record.State == models.DatabaseDeleting {
		return errors.New(DatabaseNotReady)
	}
	if payload.Phase == "applying" {
		if err := backend.Delete(fresh, record); err != nil {
			return err
		}
		record.BackendOID = 0
		record.ClusterID = ""
		payload.Phase = ""
		if err := database.DB.WithContext(ctx).Model(record).Updates(map[string]any{"backend_oid": 0, "cluster_id": "", "state": models.DatabasePending}).Error; err != nil {
			return err
		}
		if err := savePhase(ctx, operation, payload); err != nil {
			return err
		}
	}
	if payload.Phase != "verify" {
		if operation.Kind == "clone" && record.Kind() == models.EngineLibSQL {
			payload.Phase = "applying"
			if err := savePhase(ctx, operation, payload); err != nil {
				return err
			}
			if err := backend.Fork(fresh, source, record, payload.Moment); err != nil {
				return err
			}
		} else {
			recoveringCreation := payload.Phase == "creating"
			payload.Phase = "creating"
			if err := savePhase(ctx, operation, payload); err != nil {
				return err
			}
			existing := false
			if recoveringCreation && record.Kind() == models.EngineLibSQL {
				existing, err = sqld.NamespaceExists(fresh, record.Name)
				if err != nil {
					return err
				}
			}
			if !existing {
				if err := backend.Create(fresh, record); err != nil {
					return err
				}
			}
			if err := database.DB.WithContext(ctx).Model(record).Updates(map[string]any{"backend_oid": record.BackendOID, "cluster_id": record.ClusterID, "state": models.DatabaseApplying}).Error; err != nil {
				return err
			}
			record.State = models.DatabaseApplying
			payload.Phase = "applying"
			if err := savePhase(ctx, operation, payload); err != nil {
				return err
			}
			if record.Kind() == models.EnginePostgres {
				if defaultError := control.ProvisionDefaults(fresh, record); defaultError != nil {
					return defaultError
				}
			}
			authorize := func(step context.Context) error { _, _, err := operationAuthority(step, record, operation); return err }
			options := pgtransfer.Options{Authorize: authorize, Timeout: operationTimeout}
			switch operation.Kind {
			case "import":
				if record.Kind() == models.EnginePostgres {
					err = pgtransfer.ImportWithOptions(fresh, record, payload.Path, options)
				} else {
					err = replay(fresh, record.Name, payload.Path)
				}
			case "clone":
				err = pgtransfer.CloneWithOptions(fresh, source, record, options)
			case "remote-clone":
				err = pgtransfer.CloneRemoteWithOptions(fresh, origin, record, options)
			}
			if err != nil {
				return err
			}
		}
		payload.Phase = "verify"
		if err := savePhase(ctx, operation, payload); err != nil {
			return err
		}
	}
	fresh, _, err = operationAuthority(ctx, record, operation)
	if err != nil {
		return err
	}
	if record.Kind() == models.EnginePostgres {
		if err := control.Provision(fresh, record); err != nil {
			return err
		}
		if defaultError := control.ProvisionDefaults(fresh, record); defaultError != nil {
			return defaultError
		}
		if err := primarycredentials.Ensure(fresh, record, operation.AccountID); err != nil {
			return err
		}
		if err := pgmetrics.SampleDatabase(fresh, record); err != nil {
			return err
		}
	} else {
		if _, err := backend.Statistics(fresh, record); err != nil {
			return err
		}
	}
	if err := clearRemoteSecret(operation); err != nil {
		return err
	}
	err = completeOperation(ctx, record, operation)
	if err != nil {
		return err
	}
	proxy.Forget()
	if payload.Path != "" {
		os.Remove(payload.Path)
	}
	if record.Kind() == models.EngineLibSQL {
		go hosting.Publish(context.WithoutCancel(ctx), record.Name)
	}
	return nil
}

func Reconcile(ctx context.Context) error {
	var operations []models.DatabaseOperation
	if err := database.DB.WithContext(ctx).Where("kind IN ? AND state IN ? AND (lease_until IS NULL OR lease_until < ?)", []string{"create", "import", "clone", "delete", "remote-clone"}, []string{models.DatabasePending, models.DatabaseApplying, models.DatabaseFailed, remoteStagingState}, time.Now()).Order("attempts asc, id asc").Limit(25).Find(&operations).Error; err != nil {
		return err
	}
	var failures []error
	for index := range operations {
		operation := &operations[index]
		var record models.Database
		if err := database.DB.WithContext(ctx).First(&record, operation.DatabaseID).Error; err != nil {
			failures = append(failures, err)
			continue
		}
		if err := executeOperation(ctx, &record, operation); err != nil {
			failures = append(failures, errors.New(err.Message))
		}
	}
	return errors.Join(failures...)
}
