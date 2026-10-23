package databases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgtransfer"
	"sqldash/secretstore"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

const remoteSecretNamespace = "postgres-remote-source"
const remoteStagingState = "staging"

type remotePayload struct {
	Host              string
	Port              int
	Database          string
	Username          string
	SSLMode           string
	RootCertificate   []byte
	PasswordReference string
}

func CloneRemote(ctx context.Context, asked string, source pgtransfer.RemoteSource) *fiber.Error {
	fresh, actor, err := creationActor(ctx, models.EnginePostgres, permissions.DatabaseImport)
	if err != nil {
		return err
	}
	if !actor.Can(permissions.PostgresDatabaseCreate) {
		return shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	name, err := acceptableEngineName(asked, models.EnginePostgres)
	if err != nil {
		return err
	}
	options := pgtransfer.Options{Authorize: func(step context.Context) error {
		_, current, err := creationActor(step, models.EnginePostgres, permissions.DatabaseImport)
		if err != nil {
			return err
		}
		if !current.Can(permissions.PostgresDatabaseCreate) {
			return errors.New(AuthorityRefused)
		}
		return nil
	}, Timeout: 2 * time.Minute}
	if source.Password == "" {
		return shortcuts.ServiceError(http.StatusBadRequest, RemoteRefused)
	}
	if err := pgtransfer.ValidateRemoteSource(source); err != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, remoteFailureMessage(err))
	}
	manifest, inspectError := pgtransfer.InspectRemoteWithOptions(fresh, source, options)
	if inspectError != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, remoteFailureMessage(inspectError))
	}
	if needsExtensions(manifest.Extensions) {
		return shortcuts.ServiceError(http.StatusBadRequest, ExtensionsMustBeManaged)
	}
	origin := &remotePayload{Host: source.Host, Port: source.Port, Database: source.Database, Username: source.Username, SSLMode: source.SSLMode, RootCertificate: source.RootCertificate}
	record, operation, reserveError := reserveDatabase(fresh, name, models.EnginePostgres, actor.Account.ID, "remote-clone", 0, operationPayload{Remote: origin})
	if reserveError != nil {
		return reserveError
	}
	reference := strconv.FormatUint(uint64(operation.ID), 10)
	origin.PasswordReference = reference
	payload := operationPayload{Remote: origin}
	encoded, encodeError := json.Marshal(payload)
	if encodeError != nil {
		return shortcuts.ServiceError(http.StatusInternalServerError, RemoteRefused)
	}
	operation.Payload = string(encoded)
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return shortcuts.ServiceError(http.StatusServiceUnavailable, RemoteRefused)
	}
	password := []byte(source.Password)
	defer clear(password)
	if sealError := store.Set(remoteSecretNamespace, reference, password); sealError != nil {
		return shortcuts.ServiceError(http.StatusServiceUnavailable, RemoteRefused)
	}
	prepared := database.DB.WithContext(fresh).Model(operation).Where("state = ?", remoteStagingState).Updates(map[string]any{"payload": operation.Payload, "state": models.DatabasePending, "lease_until": nil})
	if prepared.Error != nil || prepared.RowsAffected != 1 {
		return shortcuts.ServiceError(http.StatusInternalServerError, RemoteRefused)
	}
	operation.State = models.DatabasePending
	operation.LeaseUntil = nil
	return executeOperation(fresh, record, operation)
}

func remoteSource(ctx context.Context, record *models.Database, operation *models.DatabaseOperation, payload *operationPayload) (pgtransfer.RemoteSource, error) {
	if payload.Remote == nil || payload.Remote.PasswordReference != strconv.FormatUint(uint64(operation.ID), 10) {
		return pgtransfer.RemoteSource{}, errors.New(RemoteRefused)
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return pgtransfer.RemoteSource{}, err
	}
	password, err := store.Get(remoteSecretNamespace, payload.Remote.PasswordReference)
	if err != nil {
		return pgtransfer.RemoteSource{}, err
	}
	defer clear(password)
	source := pgtransfer.RemoteSource{Host: payload.Remote.Host, Port: payload.Remote.Port, Database: payload.Remote.Database, Username: payload.Remote.Username, Password: string(password), SSLMode: payload.Remote.SSLMode, RootCertificate: payload.Remote.RootCertificate}
	options := pgtransfer.Options{Authorize: func(step context.Context) error { _, _, err := operationAuthority(step, record, operation); return err }, Timeout: 2 * time.Minute}
	manifest, err := pgtransfer.InspectRemoteWithOptions(ctx, source, options)
	if err != nil {
		return pgtransfer.RemoteSource{}, err
	}
	if needsExtensions(manifest.Extensions) {
		return pgtransfer.RemoteSource{}, errors.New(ExtensionsMustBeManaged)
	}
	return source, nil
}

func clearRemoteSecret(operation *models.DatabaseOperation) error {
	if operation.Kind != "remote-clone" {
		return nil
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return err
	}
	return store.Delete(remoteSecretNamespace, strconv.FormatUint(uint64(operation.ID), 10))
}

func needsExtensions(extensions []pgtransfer.RemoteExtension) bool {
	for _, extension := range extensions {
		if extension.Name != "plpgsql" && extension.Name != "pg_stat_statements" {
			return true
		}
	}
	return false
}
