package control

import (
	"context"
	"time"

	"sqldash/models"
)

func VerifyLoginPassword(ctx context.Context, record *models.Database, credential *models.PostgresConnection, password string) error {
	username, identityError := validateLogin(record, credential)
	if identityError != nil {
		return identityError
	}
	if !credential.Usable(time.Now()) || password == "" {
		return InvalidConnection
	}
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	administrator, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(administrator)
	if lockError := acquireSessionLock(ctx, administrator, record); lockError != nil {
		return lockError
	}
	if verifyError := verifyDatabase(ctx, administrator, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	identity, roleError := verifiedSessionRole(ctx, administrator, record, roles, username)
	if roleError != nil {
		return roleError
	}
	configuration, configurationError := connectionConfiguration(record.Name, username, password)
	if configurationError != nil {
		return configurationError
	}
	configuration.RuntimeParams["default_transaction_read_only"] = "on"
	connection, authenticationError := authenticatedConnect(ctx, configuration)
	if authenticationError != nil {
		return authenticationError
	}
	defer closeConnection(connection)
	return validateBackendPID(ctx, administrator, connection, record, roles, identifier, identity)
}
