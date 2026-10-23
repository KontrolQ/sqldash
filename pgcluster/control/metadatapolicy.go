package control

import (
	"context"
	"strconv"

	"sqldash/config"
	"sqldash/models"
	"sqldash/secretstore"

	"github.com/jackc/pgx/v5"
)

var metadataCatalogRelations = []string{
	"pg_namespace", "pg_class", "pg_attribute", "pg_attrdef", "pg_constraint", "pg_index", "pg_trigger",
	"pg_proc", "pg_sequence", "pg_depend", "pg_inherits", "pg_type", "pg_collation",
}

var metadataCatalogHelpers = []string{
	"format", "format_type", "unnest", "generate_series", "pg_get_expr", "pg_get_constraintdef", "pg_get_indexdef",
	"pg_get_triggerdef", "pg_get_function_identity_arguments", "pg_get_functiondef", "pg_get_viewdef", "pg_get_partkeydef",
}

func reconcileMetadata(ctx context.Context, administrator *pgx.Conn, record *models.Database, roles Roles) error {
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return Unavailable
	}
	transaction, beginError := administrator.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if roleError := provisionRole(ctx, transaction, store, record, roles.Observer, true); roleError != nil {
		return roleError
	}
	if configureError := configureMetadataRole(ctx, transaction, roles); configureError != nil {
		return configureError
	}
	if policyError := applyMetadataPolicy(ctx, transaction, record, roles); policyError != nil {
		return policyError
	}
	return SafeError(transaction.Commit(ctx))
}

func applyMetadataPolicy(ctx context.Context, transaction pgx.Tx, record *models.Database, roles Roles) error {
	observer := pgIdentifier(roles.Observer)
	databases, queryError := queryMetadataPolicyRows(ctx, transaction, "SELECT datname FROM pg_catalog.pg_database ORDER BY datname", pgx.RowTo[string])
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, name := range databases {
		if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON DATABASE "+pgIdentifier(name)+" FROM "+observer); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	schemas, queryError := queryMetadataPolicyRows(ctx, transaction, `SELECT nspname FROM pg_catalog.pg_namespace
		WHERE nspname !~ '^pg_(toast_)?temp_' ORDER BY nspname`, pgx.RowTo[string])
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, name := range schemas {
		identifier := pgIdentifier(name)
		for _, command := range []string{
			"REVOKE ALL ON SCHEMA " + identifier + " FROM " + observer,
			"REVOKE ALL ON ALL TABLES IN SCHEMA " + identifier + " FROM " + observer,
			"REVOKE ALL ON ALL SEQUENCES IN SCHEMA " + identifier + " FROM " + observer,
			"REVOKE ALL ON ALL ROUTINES IN SCHEMA " + identifier + " FROM " + observer,
		} {
			if _, revokeError := transaction.Exec(ctx, command); revokeError != nil {
				return SafeError(revokeError)
			}
		}
	}
	if revokeError := revokeMetadataObjectGrants(ctx, transaction, roles); revokeError != nil {
		return revokeError
	}
	if revokeError := revokeMetadataDefaults(ctx, transaction, roles); revokeError != nil {
		return revokeError
	}
	for _, command := range []string{
		"GRANT CONNECT ON DATABASE " + pgIdentifier(record.Name) + " TO " + observer,
		"GRANT USAGE ON SCHEMA pg_catalog TO " + observer,
	} {
		if _, grantError := transaction.Exec(ctx, command); grantError != nil {
			return SafeError(grantError)
		}
	}
	for _, name := range metadataCatalogRelations {
		if _, grantError := transaction.Exec(ctx, "GRANT SELECT ON TABLE "+pgx.Identifier{"pg_catalog", name}.Sanitize()+" TO "+observer); grantError != nil {
			return SafeError(grantError)
		}
	}
	functions, queryError := queryMetadataPolicyRows(ctx, transaction, `SELECT routine.proname, pg_catalog.pg_get_function_identity_arguments(routine.oid)
		FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE namespace.nspname = 'pg_catalog' AND routine.proname = ANY($1::text[]) ORDER BY routine.oid`, func(row pgx.CollectableRow) (string, error) {
		var name, arguments string
		if scanError := row.Scan(&name, &arguments); scanError != nil {
			return "", scanError
		}
		return pgx.Identifier{"pg_catalog", name}.Sanitize() + "(" + arguments + ")", nil
	}, metadataCatalogHelpers)
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, function := range functions {
		if _, grantError := transaction.Exec(ctx, "GRANT EXECUTE ON FUNCTION "+function+" TO "+observer); grantError != nil {
			return SafeError(grantError)
		}
	}
	return verifyMetadataIsolation(ctx, transaction, record, roles)
}

func revokeMetadataObjectGrants(ctx context.Context, transaction pgx.Tx, roles Roles) error {
	commands, queryError := queryMetadataPolicyRows(ctx, transaction, `SELECT DISTINCT namespace.nspname, relation.relname, attribute.attname
		FROM pg_catalog.pg_attribute AS attribute JOIN pg_catalog.pg_class AS relation ON relation.oid = attribute.attrelid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace,
		LATERAL pg_catalog.aclexplode(attribute.attacl) AS privilege
		WHERE privilege.grantee = (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = $1)`, func(row pgx.CollectableRow) (string, error) {
		var schema, relation, column string
		if scanError := row.Scan(&schema, &relation, &column); scanError != nil {
			return "", scanError
		}
		return "REVOKE ALL (" + pgIdentifier(column) + ") ON TABLE " + pgx.Identifier{schema, relation}.Sanitize() + " FROM " + pgIdentifier(roles.Observer), nil
	}, roles.Observer)
	if queryError != nil {
		return SafeError(queryError)
	}
	largeObjectCommands, queryError := queryMetadataPolicyRows(ctx, transaction, `SELECT oid FROM pg_catalog.pg_largeobject_metadata
		WHERE pg_catalog.has_largeobject_privilege($1, oid, 'SELECT,UPDATE')`, func(row pgx.CollectableRow) (string, error) {
		var identifier uint32
		if scanError := row.Scan(&identifier); scanError != nil {
			return "", scanError
		}
		return "REVOKE ALL ON LARGE OBJECT " + strconv.FormatUint(uint64(identifier), 10) + " FROM " + pgIdentifier(roles.Observer), nil
	}, roles.Observer)
	if queryError != nil {
		return SafeError(queryError)
	}
	commands = append(commands, largeObjectCommands...)
	for _, command := range commands {
		if _, revokeError := transaction.Exec(ctx, command); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	return nil
}

func revokeMetadataDefaults(ctx context.Context, transaction pgx.Tx, roles Roles) error {
	type defaultScope struct {
		Prefix string
		Schema string
	}
	scopes, queryError := queryMetadataPolicyRows(ctx, transaction, `SELECT DISTINCT creator.rolname, COALESCE(namespace.nspname, '')
		FROM pg_catalog.pg_default_acl AS defaults JOIN pg_catalog.pg_roles AS creator ON creator.oid = defaults.defaclrole
		LEFT JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = defaults.defaclnamespace,
		LATERAL pg_catalog.aclexplode(defaults.defaclacl) AS privilege
		WHERE privilege.grantee = (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = $1)`, func(row pgx.CollectableRow) (defaultScope, error) {
		var creator, schema string
		if scanError := row.Scan(&creator, &schema); scanError != nil {
			return defaultScope{}, scanError
		}
		prefix := "ALTER DEFAULT PRIVILEGES FOR ROLE " + pgIdentifier(creator)
		if schema != "" {
			prefix += " IN SCHEMA " + pgIdentifier(schema)
		}
		return defaultScope{Prefix: prefix, Schema: schema}, nil
	}, roles.Observer)
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, scope := range scopes {
		objects := []string{"TABLES", "SEQUENCES", "FUNCTIONS", "TYPES"}
		if scope.Schema == "" {
			objects = append(objects, "SCHEMAS")
		}
		for _, object := range objects {
			if _, revokeError := transaction.Exec(ctx, scope.Prefix+" REVOKE ALL ON "+object+" FROM "+pgIdentifier(roles.Observer)); revokeError != nil {
				return SafeError(revokeError)
			}
		}
	}
	return nil
}

func queryMetadataPolicyRows[Value any](ctx context.Context, transaction pgx.Tx, statement string, collect pgx.RowToFunc[Value], arguments ...any) ([]Value, error) {
	rows, queryError := transaction.Query(ctx, statement, arguments...)
	if queryError != nil {
		return nil, queryError
	}
	return pgx.CollectRows(rows, collect)
}
