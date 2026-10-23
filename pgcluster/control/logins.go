package control

import (
	"context"
	"errors"
	"time"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

func NativeLogin(ctx context.Context, record *models.Database, connection *models.PostgresConnection, password string) error {
	if password == "" {
		return InvalidPassword
	}
	return reconcileLogin(ctx, record, connection, password)
}

func RotateLogin(ctx context.Context, record *models.Database, connection *models.PostgresConnection, password string) error {
	if password == "" {
		return InvalidPassword
	}
	if rotateError := reconcileLogin(ctx, record, connection, password); rotateError != nil {
		return rotateError
	}
	return terminateLogin(ctx, record, connection.Username)
}

func ReconcileLogin(ctx context.Context, record *models.Database, connection *models.PostgresConnection, password string) error {
	if connection != nil && (connection.RevokedAt != nil || connection.ExpiresAt != nil && !time.Now().Before(*connection.ExpiresAt)) {
		return RevokeLogin(ctx, record, connection)
	}
	return reconcileLogin(ctx, record, connection, password)
}

func validateLogin(record *models.Database, connection *models.PostgresConnection) (string, error) {
	username, identityError := ApplicationUsername(record, connection)
	if identityError != nil {
		return "", identityError
	}
	if connection.Username != username || connection.Scope != models.ScopeReadOnly && connection.Scope != models.ScopeReadWrite {
		return "", InvalidConnection
	}
	return username, nil
}

func reconcileLogin(ctx context.Context, record *models.Database, credential *models.PostgresConnection, password string) error {
	username, identityError := validateLogin(record, credential)
	if identityError != nil {
		return identityError
	}
	if credential.RevokedAt != nil || credential.ExpiresAt != nil && !time.Now().Before(*credential.ExpiresAt) {
		return InvalidConnection
	}
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		return lockError
	}
	if password == "" {
		var exists bool
		if queryError := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = $1)", username).Scan(&exists); queryError != nil {
			return SafeError(queryError)
		}
		if !exists {
			return InvalidPassword
		}
	}
	settings, settingsError := databaseSettings(ctx, connection, record)
	if settingsError != nil {
		return settingsError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if ensureError := ensureRole(ctx, transaction, record, username, true, password); ensureError != nil {
		return ensureError
	}
	group, readOnly := roles.ReadAccess, "on"
	if credential.Scope == models.ScopeReadWrite {
		group, readOnly = roles.WriteAccess, "off"
	}
	if grantError := dataMembership(ctx, transaction, username, group); grantError != nil {
		return grantError
	}
	if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON DATABASE "+pgIdentifier(record.Name)+" FROM "+pgIdentifier(username)); revokeError != nil {
		return SafeError(revokeError)
	}
	for setting, value := range map[string]string{
		"default_transaction_read_only":       readOnly,
		"statement_timeout":                   "30000",
		"idle_in_transaction_session_timeout": "30000",
		"lock_timeout":                        "5000",
	} {
		if _, settingError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(username)+" SET "+setting+" = "+pgLiteral(value)); settingError != nil {
			return SafeError(settingError)
		}
	}
	if _, resetError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(username)+" IN DATABASE "+pgIdentifier(record.Name)+" RESET ALL"); resetError != nil {
		return SafeError(resetError)
	}
	if settingError := applySessionSettings(ctx, transaction, record, username, settings); settingError != nil {
		return settingError
	}
	if credential.ExpiresAt != nil {
		if _, expiryError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(username)+" VALID UNTIL "+pgLiteral(credential.ExpiresAt.UTC().Format(time.RFC3339Nano))); expiryError != nil {
			return SafeError(expiryError)
		}
	}
	if commitError := transaction.Commit(ctx); commitError != nil {
		return SafeError(commitError)
	}
	closeConnection(connection)
	return ReconcilePolicy(ctx, record)
}

func RevokeLogin(ctx context.Context, record *models.Database, credential *models.PostgresConnection) error {
	username, identityError := validateLogin(record, credential)
	if identityError != nil {
		return identityError
	}
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		return lockError
	}
	var marker string
	queryError := connection.QueryRow(ctx, `SELECT COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), '') FROM pg_catalog.pg_roles WHERE rolname = $1`, username).Scan(&marker)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return nil
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	if marker != roleMarker(record, username) {
		return RoleMismatch
	}
	if _, revokeError := connection.Exec(ctx, "ALTER ROLE "+pgIdentifier(username)+" NOLOGIN PASSWORD NULL"); revokeError != nil {
		return SafeError(revokeError)
	}
	if _, membershipError := connection.Exec(ctx, "REVOKE "+pgIdentifier(roles.ReadAccess)+", "+pgIdentifier(roles.WriteAccess)+" FROM "+pgIdentifier(username)+" CASCADE"); membershipError != nil {
		return SafeError(membershipError)
	}
	return terminateRoleSessions(ctx, connection, record, username)
}
