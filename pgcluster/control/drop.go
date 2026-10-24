package control

import (
	"context"
	"errors"

	"sqldash/config"
	"sqldash/models"
	"sqldash/secretstore"

	"github.com/jackc/pgx/v5"
)

func Drop(ctx context.Context, record *models.Database) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	if record.ClusterID != "" && record.ClusterID != identifier {
		return IdentityMismatch
	}
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		return lockError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	applications, queryError := applicationRoleNames(ctx, transaction, record)
	if queryError != nil {
		rollbackTransaction(transaction)
		return queryError
	}
	transfers, queryError := transferRoleNames(ctx, transaction, record)
	rollbackTransaction(transaction)
	if queryError != nil {
		return queryError
	}
	names := append(applications, transfers...)
	names = append(names, []string{roles.Observer, roles.Reader, roles.Writer, roles.Owner, roles.ReadAccess, roles.WriteAccess, roles.ObjectAccess, roles.Custodian}...)
	existing := []string{}
	for _, name := range names {
		var marker string
		findError := connection.QueryRow(ctx, `SELECT COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), '') FROM pg_catalog.pg_roles WHERE rolname = $1`, name).Scan(&marker)
		if errors.Is(findError, pgx.ErrNoRows) {
			continue
		}
		if findError != nil {
			return SafeError(findError)
		}
		if marker != roleMarker(record, name) {
			return RoleMismatch
		}
		existing = append(existing, name)
	}
	var objectOID uint32
	var owner, marker string
	findError := connection.QueryRow(ctx, `SELECT database.oid, owner.rolname,
		COALESCE(pg_catalog.shobj_description(database.oid, 'pg_database'), '')
		FROM pg_catalog.pg_database AS database JOIN pg_catalog.pg_roles AS owner ON owner.oid = database.datdba WHERE database.datname = $1`, record.Name).Scan(&objectOID, &owner, &marker)
	if findError != nil && !errors.Is(findError, pgx.ErrNoRows) {
		return SafeError(findError)
	}
	if findError == nil && (owner != roles.Custodian || marker != "" && marker != databaseMarker(record) || record.BackendOID != 0 && record.BackendOID != objectOID) {
		return IdentityMismatch
	}
	for _, name := range existing {
		if _, revokeError := connection.Exec(ctx, "ALTER ROLE "+pgIdentifier(name)+" NOLOGIN PASSWORD NULL"); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	if findError == nil {
		if _, disableError := connection.Exec(ctx, "ALTER DATABASE "+pgIdentifier(record.Name)+" ALLOW_CONNECTIONS false"); disableError != nil {
			return SafeError(disableError)
		}
		if _, dropError := connection.Exec(ctx, "DROP DATABASE "+pgIdentifier(record.Name)+" WITH (FORCE)"); dropError != nil {
			return SafeError(dropError)
		}
	}
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return Unavailable
	}
	for _, name := range names {
		if containsRole(existing, name) {
			if _, dropError := connection.Exec(ctx, "DROP ROLE "+pgIdentifier(name)); dropError != nil {
				return SafeError(dropError)
			}
		}
		if deleteError := store.Delete(SecretNamespace, name); deleteError != nil {
			return Unavailable
		}
	}
	return nil
}

func containsRole(roles []string, name string) bool {
	for _, role := range roles {
		if role == name {
			return true
		}
	}
	return false
}

func rollbackTransaction(transaction pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), CleanupTimeout)
	defer cancel()
	transaction.Rollback(ctx)
}
