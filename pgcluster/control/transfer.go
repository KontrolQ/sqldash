package control

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"sqldash/config"
	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

type TransferCredential struct {
	Address  string
	Database string
	Username string
	Password string
}

func TransferLogin(ctx context.Context, record *models.Database, mode SessionMode) (*TransferCredential, error) {
	if mode != ReadOnly && mode != Objects {
		return nil, InvalidConnection
	}
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return nil, roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return nil, connectError
	}
	defer closeConnection(connection)
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return nil, verifyError
	}
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		return nil, lockError
	}
	identity := make([]byte, 12)
	if _, randomError := rand.Read(identity); randomError != nil {
		return nil, Unavailable
	}
	username := strings.TrimSuffix(roles.Custodian, "custodian") + "transfer_" + hex.EncodeToString(identity)
	password, passwordError := GeneratePassword()
	if passwordError != nil {
		return nil, Unavailable
	}
	settings, settingsError := databaseSettings(ctx, connection, record)
	if settingsError != nil {
		return nil, settingsError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return nil, SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if ensureError := ensureRole(ctx, transaction, record, username, true, password); ensureError != nil {
		return nil, ensureError
	}
	group, readOnly := roles.ReadAccess, "on"
	if mode == Objects {
		group, readOnly = roles.Owner, "off"
	}
	if grantError := dataMembership(ctx, transaction, username, group); grantError != nil {
		return nil, grantError
	}
	if _, grantError := transaction.Exec(ctx, "GRANT CONNECT ON DATABASE "+pgIdentifier(record.Name)+" TO "+pgIdentifier(username)); grantError != nil {
		return nil, SafeError(grantError)
	}
	if _, settingError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(username)+" SET default_transaction_read_only = "+pgLiteral(readOnly)); settingError != nil {
		return nil, SafeError(settingError)
	}
	if settingError := applySessionSettings(ctx, transaction, record, username, settings); settingError != nil {
		return nil, settingError
	}
	credential := &TransferCredential{Address: config.Postgres.Address, Database: record.Name, Username: username, Password: password}
	if commitError := transaction.Commit(ctx); commitError != nil {
		closeConnection(connection)
		cleanupContext, cancel := context.WithTimeout(context.Background(), StatementTimeout)
		defer cancel()
		if cleanupError := FinishTransfer(cleanupContext, record, username, false); cleanupError != nil {
			credential.Password = ""
			return credential, TerminationUnconfirmed
		}
		return nil, SafeError(commitError)
	}
	return credential, nil
}

func FinishTransfer(ctx context.Context, record *models.Database, username string, success bool) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	prefix := strings.TrimSuffix(roles.Custodian, "custodian") + "transfer_"
	if !strings.HasPrefix(username, prefix) || len(username) != len(prefix)+24 || !validIdentifier(username) {
		return RoleMismatch
	}
	if _, decodeError := hex.DecodeString(strings.TrimPrefix(username, prefix)); decodeError != nil {
		return RoleMismatch
	}
	connection, identifier, connectError := privateAdminConnect(ctx, record.Name)
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
		return reconcilePolicy(ctx, connection, record, roles)
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	if marker != roleMarker(record, username) {
		return RoleMismatch
	}
	if _, disableError := connection.Exec(ctx, "ALTER ROLE "+pgIdentifier(username)+" NOLOGIN PASSWORD NULL"); disableError != nil {
		return SafeError(disableError)
	}
	if terminateError := terminateRoleSessions(ctx, connection, record, username); terminateError != nil {
		return terminateError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if success {
		if extensionError := validateExtensions(ctx, connection); extensionError != nil {
			return extensionError
		}
		if _, ownershipError := transaction.Exec(ctx, "REASSIGN OWNED BY "+pgIdentifier(username)+" TO "+pgIdentifier(roles.Owner)); ownershipError != nil {
			return SafeError(ownershipError)
		}
	}
	if _, cleanupError := transaction.Exec(ctx, "DROP OWNED BY "+pgIdentifier(username)); cleanupError != nil {
		return SafeError(cleanupError)
	}
	if _, dropError := transaction.Exec(ctx, "DROP ROLE "+pgIdentifier(username)); dropError != nil {
		return SafeError(dropError)
	}
	if commitError := transaction.Commit(ctx); commitError != nil {
		return SafeError(commitError)
	}
	if roleError := provisionRoles(ctx, connection, record, roles); roleError != nil {
		return roleError
	}
	return reconcilePolicy(ctx, connection, record, roles)
}
