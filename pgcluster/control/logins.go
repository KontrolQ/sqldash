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
	if credential.RevokedAt != nil || credential.ExpiresAt != nil && !time.Now().Before(*credential.ExpiresAt) || credential.PrimaryConnection && credential.Scope != models.ScopeReadWrite {
		return InvalidConnection
	}
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, cleanup, connectError := loginAdministration(ctx, record, roles)
	if connectError != nil {
		return connectError
	}
	defer cleanup()
	if password == "" {
		var exists bool
		if queryError := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = $1)", username).Scan(&exists); queryError != nil {
			return SafeError(queryError)
		}
		if !exists {
			return InvalidPassword
		}
	}
	if !credential.PrimaryConnection {
		if _, retireError := retireLogin(ctx, connection, record, username, false); retireError != nil {
			return retireError
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
	if credential.PrimaryConnection {
		if grantError := dataMembership(ctx, transaction, username, roles.ObjectAccess); grantError != nil {
			return grantError
		}
	} else if _, ownershipError := transaction.Exec(ctx, "REASSIGN OWNED BY "+pgIdentifier(username)+" TO "+pgIdentifier(roles.ObjectAccess)); ownershipError != nil {
		return SafeError(ownershipError)
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
	if policyError := applyPolicy(ctx, transaction, record, roles); policyError != nil {
		return policyError
	}
	return SafeError(transaction.Commit(ctx))
}

func RevokeLogin(ctx context.Context, record *models.Database, credential *models.PostgresConnection) error {
	return retireCredential(ctx, record, credential, true)
}

func SuspendLogin(ctx context.Context, record *models.Database, credential *models.PostgresConnection) error {
	return retireCredential(ctx, record, credential, false)
}

func retireCredential(ctx context.Context, record *models.Database, credential *models.PostgresConnection, clearPassword bool) error {
	username, identityError := validateLogin(record, credential)
	if identityError != nil {
		return identityError
	}
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, cleanup, connectError := loginAdministration(ctx, record, roles)
	if connectError != nil {
		return connectError
	}
	defer cleanup()
	exists, retireError := retireLogin(ctx, connection, record, username, clearPassword)
	if retireError != nil || !exists {
		return retireError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if _, ownershipError := transaction.Exec(ctx, "REASSIGN OWNED BY "+pgIdentifier(username)+" TO "+pgIdentifier(roles.ObjectAccess)); ownershipError != nil {
		return SafeError(ownershipError)
	}
	if _, membershipError := transaction.Exec(ctx, "REVOKE "+pgIdentifier(roles.ReadAccess)+", "+pgIdentifier(roles.WriteAccess)+", "+pgIdentifier(roles.Owner)+", "+pgIdentifier(roles.ObjectAccess)+" FROM "+pgIdentifier(username)+" CASCADE"); membershipError != nil {
		return SafeError(membershipError)
	}
	if policyError := applyPolicy(ctx, transaction, record, roles); policyError != nil {
		return policyError
	}
	return SafeError(transaction.Commit(ctx))
}

func loginAdministration(ctx context.Context, record *models.Database, roles Roles) (*pgx.Conn, func(), error) {
	lifecycle, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return nil, nil, connectError
	}
	if lockError := acquireDatabaseLock(ctx, lifecycle, record); lockError != nil {
		closeConnection(lifecycle)
		return nil, nil, lockError
	}
	if verifyError := verifyDatabase(ctx, lifecycle, record, roles, identifier, false); verifyError != nil {
		closeConnection(lifecycle)
		return nil, nil, verifyError
	}
	connection, targetIdentifier, connectError := privateAdminConnect(ctx, record.Name)
	cleanup := func() {
		closeConnection(connection)
		closeConnection(lifecycle)
	}
	if connectError != nil {
		cleanup()
		return nil, nil, connectError
	}
	if targetIdentifier != identifier {
		cleanup()
		return nil, nil, IdentityMismatch
	}
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		cleanup()
		return nil, nil, lockError
	}
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		cleanup()
		return nil, nil, verifyError
	}
	return connection, cleanup, nil
}

func verifyLoginMembership(ctx context.Context, inspector interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, credential *models.PostgresConnection, roles Roles) error {
	groups := []string{roles.ReadAccess}
	if credential.Scope == models.ScopeReadWrite {
		groups = []string{roles.WriteAccess}
	}
	if credential.PrimaryConnection {
		if credential.Scope != models.ScopeReadWrite {
			return RoleMismatch
		}
		groups = append(groups, roles.ObjectAccess)
	}
	var correct bool
	queryError := inspector.QueryRow(ctx, `SELECT count(*) = $2 AND count(DISTINCT membership.roleid) = $2 AND COALESCE(bool_and(
		parent.rolname = ANY($3::text[]) AND membership.inherit_option
		AND NOT membership.set_option AND NOT membership.admin_option), false)
		FROM pg_catalog.pg_auth_members AS membership
		JOIN pg_catalog.pg_roles AS parent ON parent.oid = membership.roleid
		JOIN pg_catalog.pg_roles AS member ON member.oid = membership.member
		WHERE member.rolname = $1`, credential.Username, len(groups), groups).Scan(&correct)
	if queryError != nil {
		return SafeError(queryError)
	}
	if !correct {
		return RoleMismatch
	}
	return nil
}

func retireLogin(ctx context.Context, connection *pgx.Conn, record *models.Database, username string, clearPassword bool) (bool, error) {
	var marker string
	queryError := connection.QueryRow(ctx, `SELECT COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), '') FROM pg_catalog.pg_roles WHERE rolname = $1`, username).Scan(&marker)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return false, nil
	}
	if queryError != nil {
		return false, SafeError(queryError)
	}
	if marker != roleMarker(record, username) {
		return false, RoleMismatch
	}
	command := "ALTER ROLE " + pgIdentifier(username) + " NOLOGIN"
	if clearPassword {
		command += " PASSWORD NULL"
	}
	if _, revokeError := connection.Exec(ctx, command); revokeError != nil {
		return false, SafeError(revokeError)
	}
	return true, terminateRoleSessions(ctx, connection, record, username)
}
