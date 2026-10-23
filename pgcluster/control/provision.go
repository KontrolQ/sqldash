package control

import (
	"context"
	"errors"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

func Provision(ctx context.Context, record *models.Database) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	if record.State == models.DatabaseDeleting || record.State == models.DatabaseReady && (record.BackendOID == 0 || record.ClusterID == "") {
		return IdentityMismatch
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
	if provisionError := provisionRoles(ctx, connection, record, roles); provisionError != nil {
		return provisionError
	}
	var objectOID uint32
	var owner, marker, encoding string
	queryError := connection.QueryRow(ctx, `SELECT database.oid, owner.rolname,
		COALESCE(pg_catalog.shobj_description(database.oid, 'pg_database'), ''), pg_catalog.pg_encoding_to_char(database.encoding)
		FROM pg_catalog.pg_database AS database JOIN pg_catalog.pg_roles AS owner ON owner.oid = database.datdba
		WHERE database.datname = $1`, record.Name).Scan(&objectOID, &owner, &marker, &encoding)
	if errors.Is(queryError, pgx.ErrNoRows) {
		if record.BackendOID != 0 {
			return IdentityMismatch
		}
		command := "CREATE DATABASE " + pgIdentifier(record.Name) + " WITH OWNER " + pgIdentifier(roles.Custodian) + " TEMPLATE template0 ENCODING 'UTF8' ALLOW_CONNECTIONS false"
		if _, createError := connection.Exec(ctx, command); createError != nil {
			return SafeError(createError)
		}
	} else if queryError != nil {
		return SafeError(queryError)
	} else if owner != roles.Custodian || marker != "" && marker != databaseMarker(record) || encoding != "UTF8" || record.BackendOID != 0 && record.BackendOID != objectOID {
		return IdentityMismatch
	}
	if _, commentError := connection.Exec(ctx, "COMMENT ON DATABASE "+pgIdentifier(record.Name)+" IS "+pgLiteral(databaseMarker(record))); commentError != nil {
		return SafeError(commentError)
	}
	if _, revokeError := connection.Exec(ctx, "REVOKE ALL ON DATABASE "+pgIdentifier(record.Name)+" FROM PUBLIC"); revokeError != nil {
		return SafeError(revokeError)
	}
	if _, enableError := connection.Exec(ctx, "ALTER DATABASE "+pgIdentifier(record.Name)+" ALLOW_CONNECTIONS true"); enableError != nil {
		return SafeError(enableError)
	}
	target, targetIdentifier, targetError := privateAdminConnect(ctx, record.Name)
	if targetError != nil {
		return targetError
	}
	defer closeConnection(target)
	if identifier != targetIdentifier {
		return IdentityMismatch
	}
	if verifyError := verifyDatabase(ctx, target, record, roles, identifier, true); verifyError != nil {
		return verifyError
	}
	if policyError := reconcilePolicy(ctx, target, record, roles); policyError != nil {
		return policyError
	}
	if queryError := target.QueryRow(ctx, "SELECT oid FROM pg_catalog.pg_database WHERE datname = current_database()").Scan(&objectOID); queryError != nil {
		return SafeError(queryError)
	}
	record.BackendOID = objectOID
	record.ClusterID = identifier
	return nil
}
