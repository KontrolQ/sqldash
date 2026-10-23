package control

import (
	"context"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

func configureMetadataRole(ctx context.Context, transaction pgx.Tx, roles Roles) error {
	rows, queryError := transaction.Query(ctx, `SELECT parent.rolname, member.rolname FROM pg_catalog.pg_auth_members AS membership
		JOIN pg_catalog.pg_roles AS parent ON parent.oid = membership.roleid
		JOIN pg_catalog.pg_roles AS member ON member.oid = membership.member
		WHERE parent.rolname = $1 OR member.rolname = $1`, roles.Observer)
	if queryError != nil {
		return SafeError(queryError)
	}
	memberships := [][2]string{}
	for rows.Next() {
		var parent, member string
		if scanError := rows.Scan(&parent, &member); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		memberships = append(memberships, [2]string{parent, member})
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, membership := range memberships {
		if _, revokeError := transaction.Exec(ctx, "REVOKE "+pgIdentifier(membership[0])+" FROM "+pgIdentifier(membership[1])+" CASCADE"); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	rows, queryError = transaction.Query(ctx, `SELECT database.datname FROM pg_catalog.pg_db_role_setting AS setting
		JOIN pg_catalog.pg_database AS database ON database.oid = setting.setdatabase
		JOIN pg_catalog.pg_roles AS role ON role.oid = setting.setrole WHERE role.rolname = $1`, roles.Observer)
	if queryError != nil {
		return SafeError(queryError)
	}
	databases := []string{}
	for rows.Next() {
		var name string
		if scanError := rows.Scan(&name); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		databases = append(databases, name)
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, name := range databases {
		if _, resetError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(roles.Observer)+" IN DATABASE "+pgIdentifier(name)+" RESET ALL"); resetError != nil {
			return SafeError(resetError)
		}
	}
	if _, alterError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(roles.Observer)+" NOINHERIT"); alterError != nil {
		return SafeError(alterError)
	}
	for setting, value := range map[string]string{
		"search_path":                         "pg_catalog",
		"default_transaction_read_only":       "on",
		"statement_timeout":                   "30000",
		"idle_in_transaction_session_timeout": "30000",
		"lock_timeout":                        "5000",
	} {
		if _, settingError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(roles.Observer)+" SET "+setting+" = "+pgLiteral(value)); settingError != nil {
			return SafeError(settingError)
		}
	}
	return nil
}

func ReconcileMetadata(ctx context.Context, record *models.Database) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	if record.State != models.DatabaseReady {
		return InvalidDatabase
	}
	administrator, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(administrator)
	if lockError := acquireDatabaseLock(ctx, administrator, record); lockError != nil {
		return lockError
	}
	if verifyError := verifyDatabase(ctx, administrator, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	inspector, targetIdentifier, inspectError := privateAdminConnect(ctx, record.Name)
	if inspectError != nil {
		return inspectError
	}
	defer closeConnection(inspector)
	if targetIdentifier != identifier {
		return IdentityMismatch
	}
	if lockError := acquireDatabaseLock(ctx, inspector, record); lockError != nil {
		return lockError
	}
	return reconcileMetadata(ctx, inspector, record, roles)
}
