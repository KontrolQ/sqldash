package control

import (
	"context"
	"strings"

	"sqldash/config"
	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

func ReconcilePolicy(ctx context.Context, record *models.Database) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	lifecycle, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(lifecycle)
	if lockError := acquireDatabaseLock(ctx, lifecycle, record); lockError != nil {
		return lockError
	}
	if verifyError := verifyDatabase(ctx, lifecycle, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	connection, targetIdentifier, connectError := privateAdminConnect(ctx, record.Name)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	if identifier != targetIdentifier {
		return IdentityMismatch
	}
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		return lockError
	}
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	if roleError := provisionRoles(ctx, connection, record, roles); roleError != nil {
		return roleError
	}
	return reconcilePolicy(ctx, connection, record, roles)
}

func reconcilePolicy(ctx context.Context, connection *pgx.Conn, record *models.Database, roles Roles) error {
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if policyError := applyPolicy(ctx, transaction, record, roles); policyError != nil {
		return policyError
	}
	return SafeError(transaction.Commit(ctx))
}

func applyPolicy(ctx context.Context, transaction pgx.Tx, record *models.Database, roles Roles) error {
	if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON DATABASE "+pgIdentifier(record.Name)+" FROM PUBLIC"); revokeError != nil {
		return SafeError(revokeError)
	}
	for _, name := range []string{roles.Owner, roles.Reader, roles.Writer, roles.ReadAccess, roles.WriteAccess} {
		if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON DATABASE "+pgIdentifier(record.Name)+" FROM "+pgIdentifier(name)); revokeError != nil {
			return SafeError(revokeError)
		}
		if _, grantError := transaction.Exec(ctx, "GRANT CONNECT ON DATABASE "+pgIdentifier(record.Name)+" TO "+pgIdentifier(name)); grantError != nil {
			return SafeError(grantError)
		}
	}
	var publicExists bool
	if queryError := transaction.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname = 'public')").Scan(&publicExists); queryError != nil {
		return SafeError(queryError)
	}
	if publicExists {
		if _, ownerError := transaction.Exec(ctx, "ALTER SCHEMA public OWNER TO "+pgIdentifier(roles.Owner)); ownerError != nil {
			return SafeError(ownerError)
		}
	}
	if _, grantError := transaction.Exec(ctx, "GRANT CREATE ON DATABASE "+pgIdentifier(record.Name)+" TO "+pgIdentifier(roles.Owner)); grantError != nil {
		return SafeError(grantError)
	}
	recipients, recipientError := policyRecipients(ctx, transaction, roles)
	if recipientError != nil {
		return recipientError
	}
	rows, queryError := transaction.Query(ctx, `SELECT nspname FROM pg_catalog.pg_namespace
		WHERE nspname <> 'information_schema' AND nspname !~ '^pg_' AND nspname NOT IN ($1, $2) ORDER BY nspname`, ControlSchema, ExtensionsSchema)
	if queryError != nil {
		return SafeError(queryError)
	}
	schemas := []string{}
	for rows.Next() {
		var schema string
		if scanError := rows.Scan(&schema); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		schemas = append(schemas, schema)
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, schema := range schemas {
		identifier := pgIdentifier(schema)
		commands := []string{
			"REVOKE ALL ON SCHEMA " + identifier + " FROM " + recipients,
			"GRANT USAGE ON SCHEMA " + identifier + " TO " + pgIdentifier(roles.ReadAccess) + ", " + pgIdentifier(roles.WriteAccess),
			"REVOKE ALL ON ALL TABLES IN SCHEMA " + identifier + " FROM " + recipients,
			"REVOKE ALL ON ALL SEQUENCES IN SCHEMA " + identifier + " FROM " + recipients,
			"REVOKE ALL ON ALL ROUTINES IN SCHEMA " + identifier + " FROM " + recipients,
			"GRANT SELECT ON ALL TABLES IN SCHEMA " + identifier + " TO " + pgIdentifier(roles.ReadAccess),
			"GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA " + identifier + " TO " + pgIdentifier(roles.WriteAccess),
			"GRANT SELECT ON ALL SEQUENCES IN SCHEMA " + identifier + " TO " + pgIdentifier(roles.ReadAccess),
			"GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA " + identifier + " TO " + pgIdentifier(roles.WriteAccess),
			"GRANT EXECUTE ON ALL ROUTINES IN SCHEMA " + identifier + " TO " + pgIdentifier(roles.WriteAccess),
		}
		for _, command := range commands {
			if _, policyError := transaction.Exec(ctx, command); policyError != nil {
				return SafeError(policyError)
			}
		}
		for _, creator := range []string{roles.Owner, config.Postgres.AdminUsername} {
			prefix := "ALTER DEFAULT PRIVILEGES FOR ROLE " + pgIdentifier(creator) + " IN SCHEMA " + identifier
			defaults := []string{
				prefix + " REVOKE ALL ON TABLES FROM " + recipients,
				prefix + " REVOKE ALL ON SEQUENCES FROM " + recipients,
				prefix + " REVOKE ALL ON FUNCTIONS FROM " + recipients,
				prefix + " REVOKE ALL ON TYPES FROM " + recipients,
				prefix + " GRANT USAGE ON TYPES TO " + pgIdentifier(roles.ReadAccess) + ", " + pgIdentifier(roles.WriteAccess),
				prefix + " GRANT SELECT ON TABLES TO " + pgIdentifier(roles.ReadAccess),
				prefix + " GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO " + pgIdentifier(roles.WriteAccess),
				prefix + " GRANT SELECT ON SEQUENCES TO " + pgIdentifier(roles.ReadAccess),
				prefix + " GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO " + pgIdentifier(roles.WriteAccess),
				prefix + " GRANT EXECUTE ON FUNCTIONS TO " + pgIdentifier(roles.WriteAccess),
			}
			for _, command := range defaults {
				if _, defaultError := transaction.Exec(ctx, command); defaultError != nil {
					return SafeError(defaultError)
				}
			}
		}
	}
	for _, creator := range []string{roles.Owner, config.Postgres.AdminUsername} {
		prefix := "ALTER DEFAULT PRIVILEGES FOR ROLE " + pgIdentifier(creator)
		defaults := []string{
			prefix + " REVOKE ALL ON TABLES FROM " + recipients,
			prefix + " REVOKE ALL ON SEQUENCES FROM " + recipients,
			prefix + " REVOKE ALL ON FUNCTIONS FROM " + recipients,
			prefix + " REVOKE ALL ON TYPES FROM " + recipients,
			prefix + " REVOKE ALL ON SCHEMAS FROM " + recipients,
			prefix + " GRANT SELECT ON TABLES TO " + pgIdentifier(roles.ReadAccess),
			prefix + " GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO " + pgIdentifier(roles.WriteAccess),
			prefix + " GRANT SELECT ON SEQUENCES TO " + pgIdentifier(roles.ReadAccess),
			prefix + " GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO " + pgIdentifier(roles.WriteAccess),
			prefix + " GRANT EXECUTE ON FUNCTIONS TO " + pgIdentifier(roles.WriteAccess),
			prefix + " GRANT USAGE ON TYPES TO " + pgIdentifier(roles.ReadAccess) + ", " + pgIdentifier(roles.WriteAccess),
			prefix + " GRANT USAGE ON SCHEMAS TO " + pgIdentifier(roles.ReadAccess) + ", " + pgIdentifier(roles.WriteAccess),
		}
		for _, command := range defaults {
			if _, defaultError := transaction.Exec(ctx, command); defaultError != nil {
				return SafeError(defaultError)
			}
		}
	}
	privateRecipients := recipients + ", " + pgIdentifier(roles.Owner) + ", " + pgIdentifier(roles.Custodian)
	if schemaError := ensureExtensionsSchema(ctx, transaction); schemaError != nil {
		return schemaError
	}
	if guardError := guardExtensions(ctx, transaction, privateRecipients); guardError != nil {
		return guardError
	}
	if dropGuardError := guardExtensionDrops(ctx, transaction, privateRecipients); dropGuardError != nil {
		return dropGuardError
	}
	if protectError := protectStatistics(ctx, transaction, privateRecipients); protectError != nil {
		return protectError
	}
	if extensionError := protectExtensions(ctx, transaction, roles, recipients); extensionError != nil {
		return extensionError
	}
	if routineError := protectExtensionRoutines(ctx, transaction, roles, recipients); routineError != nil {
		return routineError
	}
	if relationError := protectExtensionRelations(ctx, transaction, roles, recipients); relationError != nil {
		return relationError
	}
	if typeError := protectExtensionTypes(ctx, transaction, roles, recipients); typeError != nil {
		return typeError
	}
	if largeObjectError := protectLargeObjects(ctx, transaction, roles, recipients); largeObjectError != nil {
		return largeObjectError
	}
	return applyMetadataPolicy(ctx, transaction, record, roles)
}

func protectStatistics(ctx context.Context, transaction pgx.Tx, recipients string) error {
	rows, queryError := transaction.Query(ctx, `SELECT namespace.nspname, relation.relname
		FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE relation.relkind IN ('v', 'm') AND relation.relname LIKE 'pg_stat_%'`)
	if queryError != nil {
		return SafeError(queryError)
	}
	views := []pgx.Identifier{}
	for rows.Next() {
		var schema, name string
		if scanError := rows.Scan(&schema, &name); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		views = append(views, pgx.Identifier{schema, name})
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, view := range views {
		if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON "+view.Sanitize()+" FROM "+recipients); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	rows, queryError = transaction.Query(ctx, `SELECT namespace.nspname, routine.proname, pg_catalog.pg_get_function_identity_arguments(routine.oid)
		FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE routine.proname LIKE 'pg_stat_%'`)
	if queryError != nil {
		return SafeError(queryError)
	}
	functions := []string{}
	for rows.Next() {
		var schema, name, arguments string
		if scanError := rows.Scan(&schema, &name, &arguments); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		functions = append(functions, pgx.Identifier{schema, name}.Sanitize()+"("+arguments+")")
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, function := range functions {
		if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON FUNCTION "+function+" FROM "+recipients); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	return nil
}

func policyRecipients(ctx context.Context, transaction pgx.Tx, roles Roles) (string, error) {
	rows, queryError := transaction.Query(ctx, "SELECT rolname FROM pg_catalog.pg_roles WHERE rolname <> ALL($1::text[]) ORDER BY rolname", []string{roles.Owner, roles.Custodian, config.Postgres.AdminUsername})
	if queryError != nil {
		return "", SafeError(queryError)
	}
	defer rows.Close()
	recipients := []string{"PUBLIC"}
	for rows.Next() {
		var name string
		if scanError := rows.Scan(&name); scanError != nil {
			return "", SafeError(scanError)
		}
		recipients = append(recipients, pgIdentifier(name))
	}
	return strings.Join(recipients, ", "), SafeError(rows.Err())
}

func applicationRoleNames(ctx context.Context, transaction pgx.Tx, record *models.Database) ([]string, error) {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return nil, roleError
	}
	prefix := strings.TrimSuffix(roles.Custodian, "custodian") + "app_"
	return scopedRoleNames(ctx, transaction, record, prefix, true)
}

func transferRoleNames(ctx context.Context, transaction pgx.Tx, record *models.Database) ([]string, error) {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return nil, roleError
	}
	return scopedRoleNames(ctx, transaction, record, strings.TrimSuffix(roles.Custodian, "custodian")+"transfer_", false)
}

func scopedRoleNames(ctx context.Context, transaction pgx.Tx, record *models.Database, prefix string, includePrimary bool) ([]string, error) {
	rows, queryError := transaction.Query(ctx, `SELECT rolname FROM pg_catalog.pg_roles
		WHERE (starts_with(rolname, $1) OR ($3 AND rolname = $4)) AND pg_catalog.shobj_description(oid, 'pg_authid') = $2 || rolname ORDER BY rolname`, prefix, databaseMarker(record)+":", includePrimary, record.Name)
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if scanError := rows.Scan(&name); scanError != nil {
			return nil, SafeError(scanError)
		}
		names = append(names, name)
	}
	return names, SafeError(rows.Err())
}
