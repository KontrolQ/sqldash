package control

import (
	"context"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

type metadataInspector interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func verifyMetadataIsolation(ctx context.Context, administrator metadataInspector, record *models.Database, roles Roles) error {
	var isolated bool
	queryError := administrator.QueryRow(ctx, `SELECT NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_auth_members AS membership JOIN pg_catalog.pg_roles AS role
		ON role.oid IN (membership.member, membership.roleid) WHERE role.rolname = $1
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_shdepend AS dependency JOIN pg_catalog.pg_roles AS role ON role.oid = dependency.refobjid
		WHERE dependency.refclassid = 'pg_catalog.pg_authid'::regclass AND role.rolname = $1 AND dependency.deptype = 'o'
	) AND EXISTS (
		SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = $1 AND rolcanlogin AND NOT rolinherit
		AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
		AND pg_catalog.shobj_description(oid, 'pg_authid') = $2
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_parameter_acl AS parameter, LATERAL pg_catalog.aclexplode(parameter.paracl) AS privilege
		WHERE privilege.grantee IN (0, (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = $1))
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_default_acl AS defaults, LATERAL pg_catalog.aclexplode(defaults.defaclacl) AS privilege
		WHERE privilege.grantee = (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = $1)
	)`, roles.Observer, roleMarker(record, roles.Observer)).Scan(&isolated)
	if queryError != nil {
		return SafeError(queryError)
	}
	if !isolated {
		return PolicyUnconfirmed
	}
	queryError = administrator.QueryRow(ctx, `SELECT current_database() = $2 AND EXISTS (
		SELECT 1 FROM pg_catalog.pg_database WHERE datname = $2 AND ($3::oid = 0 OR oid = $3)
		AND pg_catalog.has_database_privilege($1, oid, 'CONNECT')
		AND NOT pg_catalog.has_database_privilege($1, oid, 'CREATE,TEMPORARY')
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_database WHERE datname <> $2 AND pg_catalog.has_database_privilege($1, oid, 'CONNECT,CREATE,TEMPORARY')
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname !~ '^pg_(toast_)?temp_'
		AND (pg_catalog.has_schema_privilege($1, oid, 'CREATE')
		OR nspname NOT IN ('pg_catalog', 'information_schema') AND pg_catalog.has_schema_privilege($1, oid, 'USAGE'))
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname NOT IN ('pg_catalog', 'information_schema') AND namespace.nspname !~ '^pg_(toast_)?temp_'
		AND CASE WHEN relation.relkind = 'S' THEN pg_catalog.has_sequence_privilege($1, relation.oid, 'USAGE,SELECT,UPDATE')
		WHEN relation.relkind IN ('r', 'p', 'v', 'm', 'f') THEN
		pg_catalog.has_table_privilege($1, relation.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
		OR pg_catalog.has_any_column_privilege($1, relation.oid, 'SELECT,INSERT,UPDATE,REFERENCES') ELSE false END
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname IN ('pg_catalog', 'information_schema') AND relation.relkind IN ('r', 'p', 'v', 'm', 'f')
		AND (pg_catalog.has_table_privilege($1, relation.oid, 'INSERT,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
		OR pg_catalog.has_any_column_privilege($1, relation.oid, 'INSERT,REFERENCES')
		OR (pg_catalog.has_table_privilege($1, relation.oid, 'UPDATE')
		OR pg_catalog.has_any_column_privilege($1, relation.oid, 'UPDATE')) AND NOT (
			namespace.nspname = 'pg_catalog' AND relation.relname = 'pg_settings' AND relation.relkind = 'v' AND relation.oid < 16384
			AND EXISTS (
				SELECT 1 FROM pg_catalog.pg_init_privs AS initial
				WHERE initial.classoid = 'pg_catalog.pg_class'::regclass AND initial.objoid = relation.oid
				AND initial.objsubid = 0 AND initial.privtype = 'i' AND initial.initprivs = relation.relacl
				AND EXISTS (
					SELECT 1 FROM pg_catalog.aclexplode(initial.initprivs) AS privilege
					WHERE privilege.grantee = 0 AND privilege.grantor = relation.relowner
					AND privilege.privilege_type = 'UPDATE' AND NOT privilege.is_grantable
				)
			) AND NOT EXISTS (
				SELECT 1 FROM pg_catalog.pg_attribute AS attribute, LATERAL pg_catalog.aclexplode(attribute.attacl) AS privilege
				WHERE attribute.attrelid = relation.oid AND privilege.privilege_type IN ('INSERT', 'UPDATE', 'REFERENCES')
			)
		))
	) AND NOT pg_catalog.has_column_privilege($1, 'pg_catalog.pg_authid', 'rolpassword', 'SELECT')
	AND NOT pg_catalog.has_column_privilege($1, 'pg_catalog.pg_subscription', 'subconninfo', 'SELECT')
	AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE (namespace.nspname NOT IN ('pg_catalog', 'information_schema') OR routine.proname LIKE 'pg_stat_%'
		OR routine.proname IN ('pg_read_file', 'pg_read_binary_file', 'pg_ls_dir'))
		AND pg_catalog.has_function_privilege($1, routine.oid, 'EXECUTE')
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_class WHERE relname LIKE 'pg_stat_%' AND relkind IN ('v', 'm')
		AND (pg_catalog.has_table_privilege($1, oid, 'SELECT') OR pg_catalog.has_any_column_privilege($1, oid, 'SELECT'))
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_largeobject_metadata WHERE pg_catalog.has_largeobject_privilege($1, oid, 'SELECT,UPDATE')
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_foreign_server WHERE pg_catalog.has_server_privilege($1, oid, 'USAGE')
	) AND NOT EXISTS (
		SELECT 1 FROM pg_catalog.pg_foreign_data_wrapper AS wrapper,
		LATERAL pg_catalog.aclexplode(wrapper.fdwacl) AS privilege
		WHERE privilege.privilege_type = 'USAGE'
		AND privilege.grantee IN (0, (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = $1))
	)`, roles.Observer, record.Name, record.BackendOID).Scan(&isolated)
	if queryError != nil {
		return SafeError(queryError)
	}
	// PUBLIC privileges cannot be denied to just this role; do not rewrite other users to hide drift.
	if !isolated {
		return PolicyUnconfirmed
	}
	var relationsReady, helpersReady bool
	queryError = administrator.QueryRow(ctx, `SELECT pg_catalog.has_schema_privilege($1, 'pg_catalog', 'USAGE') AND (
		SELECT count(*) = cardinality($2::text[]) AND bool_and(pg_catalog.has_table_privilege($1, relation.oid, 'SELECT'))
		FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname = 'pg_catalog' AND relation.relname = ANY($2::text[])
	), (
		SELECT count(DISTINCT routine.proname) = cardinality($3::text[]) AND bool_and(pg_catalog.has_function_privilege($1, routine.oid, 'EXECUTE'))
		FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE namespace.nspname = 'pg_catalog' AND routine.proname = ANY($3::text[])
	)`, roles.Observer, metadataCatalogRelations, metadataCatalogHelpers).Scan(&relationsReady, &helpersReady)
	if queryError != nil {
		return SafeError(queryError)
	}
	if !relationsReady || !helpersReady {
		return PolicyUnconfirmed
	}
	return nil
}
