package control

import (
	"context"

	"sqldash/config"
	"sqldash/models"
	"sqldash/secretstore"

	"github.com/jackc/pgx/v5"
)

func ConnectMetadata(ctx context.Context, record *models.Database) (*pgx.Conn, error) {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return nil, roleError
	}
	administrator, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return nil, connectError
	}
	defer closeConnection(administrator)
	if lockError := acquireSessionLock(ctx, administrator, record); lockError != nil {
		return nil, lockError
	}
	if verifyError := verifyDatabase(ctx, administrator, record, roles, identifier, false); verifyError != nil {
		return nil, verifyError
	}
	identity, roleError := verifiedSessionRole(ctx, administrator, record, roles, roles.Observer)
	if roleError != nil {
		return nil, roleError
	}
	inspector, targetIdentifier, inspectError := privateAdminConnect(ctx, record.Name)
	if inspectError != nil {
		return nil, inspectError
	}
	defer closeConnection(inspector)
	if targetIdentifier != identifier {
		return nil, IdentityMismatch
	}
	// Advisory locks are database-local; retain both lifecycle and target-policy locks.
	if lockError := acquireSessionLock(ctx, inspector, record); lockError != nil {
		return nil, lockError
	}
	if isolationError := verifyMetadataIsolation(ctx, inspector, record, roles); isolationError != nil {
		return nil, isolationError
	}
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return nil, Unavailable
	}
	password, passwordError := internalPassword(store, roles.Observer, false)
	if passwordError != nil {
		return nil, passwordError
	}
	configuration, configurationError := connectionConfiguration(record.Name, roles.Observer, password)
	if configurationError != nil {
		return nil, configurationError
	}
	configuration.RuntimeParams["search_path"] = "pg_catalog"
	configuration.RuntimeParams["default_transaction_read_only"] = "on"
	configuration.RuntimeParams["application_name"] = "sqldash-catalog"
	configuration.MaxProtocolMessageBodyLen = 32 << 20
	limitTransport(configuration)
	connection, authenticationError := authenticatedConnect(ctx, configuration)
	if authenticationError != nil {
		return nil, authenticationError
	}
	if verifyError := validateBackendPID(ctx, administrator, connection, record, roles, identifier, identity); verifyError != nil {
		closeConnection(connection)
		return nil, verifyError
	}
	if isolationError := verifyMetadataIsolation(ctx, inspector, record, roles); isolationError != nil {
		closeConnection(connection)
		return nil, isolationError
	}
	return connection, nil
}
