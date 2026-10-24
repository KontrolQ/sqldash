package control

import (
	"context"
	"errors"
	"strings"

	"sqldash/config"
	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

const (
	OwnershipGuard         = "sqd_object_ownership_guard"
	OwnershipGuardFunction = "guard_object_ownership"
	OwnershipFunction      = "normalize_object_ownership"
	OwnershipMarker        = "sqldash:object-ownership"
)

func guardObjectOwnership(ctx context.Context, transaction pgx.Tx, record *models.Database, roles Roles, recipients string) error {
	normalize := pgx.Identifier{ControlSchema, OwnershipFunction}.Sanitize()
	guard := pgx.Identifier{ControlSchema, OwnershipGuardFunction}.Sanitize()
	replacements := strings.NewReplacer(
		"@normalize@", normalize, "@guard@", guard,
		"@administrator@", pgLiteral(config.Postgres.AdminUsername),
		"@primary@", pgLiteral(record.Name), "@operator@", pgLiteral(roles.Owner),
		"@objects@", pgLiteral(roles.ObjectAccess), "@write@", pgLiteral(roles.WriteAccess),
		"@marker@", pgLiteral(databaseMarker(record)+":"),
		"@database_marker@", pgLiteral(databaseMarker(record)), "@custodian@", pgLiteral(roles.Custodian),
		"@ordinary@", `namespace.nspname <> 'information_schema' AND namespace.nspname !~ '^pg_'
			AND namespace.nspname NOT IN (`+pgLiteral(ControlSchema)+", "+pgLiteral(ExtensionsSchema)+`)
			AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend AS dependency
			WHERE dependency.classid = 'pg_catalog.pg_namespace'::regclass AND dependency.objid = namespace.oid
			AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.deptype = 'e')`,
	)
	body := replacements.Replace(`DECLARE creators oid[]; expected_groups text[]; authorized boolean; statement text;
		BEGIN
			IF current_database() <> @primary@ OR NOT EXISTS (
				SELECT 1 FROM pg_catalog.pg_database AS database JOIN pg_catalog.pg_roles AS owner ON owner.oid = database.datdba
				WHERE database.datname = current_database() AND owner.rolname = @custodian@
				AND pg_catalog.shobj_description(database.oid, 'pg_database') = @database_marker@
				AND pg_catalog.shobj_description(owner.oid, 'pg_authid') = @marker@ || owner.rolname
			) THEN RAISE EXCEPTION 'database identity is invalid' USING ERRCODE = '42501'; END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles AS role WHERE role.rolname = @objects@
				AND pg_catalog.shobj_description(role.oid, 'pg_authid') = @marker@ || role.rolname
				AND NOT role.rolcanlogin AND NOT role.rolsuper AND NOT role.rolcreatedb AND NOT role.rolcreaterole
				AND NOT role.rolreplication AND NOT role.rolbypassrls
				AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE member = role.oid)) THEN
				RAISE EXCEPTION 'object owner identity is invalid' USING ERRCODE = '42501';
			END IF;
			SELECT array_agg(role.oid) INTO creators FROM pg_catalog.pg_roles AS role
				WHERE role.rolname IN (@primary@, @operator@)
				AND pg_catalog.shobj_description(role.oid, 'pg_authid') = @marker@ || role.rolname
				AND NOT role.rolsuper AND NOT role.rolcreatedb AND NOT role.rolcreaterole
				AND NOT role.rolreplication AND NOT role.rolbypassrls;
			IF session_user <> @administrator@ THEN
				IF session_user = @primary@ THEN expected_groups := ARRAY[@objects@, @write@];
				ELSIF session_user = @operator@ THEN expected_groups := ARRAY[@objects@];
				ELSE RAISE EXCEPTION 'object ownership is private' USING ERRCODE = '42501'; END IF;
				SELECT count(*) = cardinality(expected_groups) AND count(DISTINCT membership.roleid) = cardinality(expected_groups)
					AND COALESCE(bool_and(parent.rolname = ANY(expected_groups) AND membership.inherit_option
					AND NOT membership.set_option AND NOT membership.admin_option), false)
					INTO authorized FROM pg_catalog.pg_auth_members AS membership
					JOIN pg_catalog.pg_roles AS parent ON parent.oid = membership.roleid
					JOIN pg_catalog.pg_roles AS member ON member.oid = membership.member
					WHERE member.rolname = session_user AND member.oid = ANY(creators) AND member.rolcanlogin;
				IF NOT authorized THEN RAISE EXCEPTION 'object creator identity is invalid' USING ERRCODE = '42501'; END IF;
			END IF;
			CASE object_class
			WHEN 'pg_catalog.pg_namespace'::regclass THEN
				SELECT pg_catalog.format('ALTER SCHEMA %I OWNER TO %I', namespace.nspname, @objects@) INTO statement
				FROM pg_catalog.pg_namespace AS namespace WHERE namespace.oid = object_id
				AND namespace.nspowner = ANY(creators) AND @ordinary@;
			WHEN 'pg_catalog.pg_class'::regclass THEN
				SELECT pg_catalog.format('ALTER %s %I.%I OWNER TO %I', CASE relation.relkind
					WHEN 'v' THEN 'VIEW' WHEN 'm' THEN 'MATERIALIZED VIEW' WHEN 'S' THEN 'SEQUENCE'
					WHEN 'f' THEN 'FOREIGN TABLE' ELSE 'TABLE' END, namespace.nspname, relation.relname, @objects@)
				INTO statement FROM pg_catalog.pg_class AS relation
				JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
				WHERE relation.oid = object_id AND relation.relowner = ANY(creators)
				AND relation.relkind IN ('r', 'p', 'v', 'm', 'S', 'f') AND @ordinary@
				AND NOT (relation.relkind = 'S' AND EXISTS (SELECT 1 FROM pg_catalog.pg_depend AS dependency
					WHERE dependency.classid = 'pg_catalog.pg_class'::regclass AND dependency.objid = relation.oid
					AND dependency.refclassid = 'pg_catalog.pg_class'::regclass AND dependency.deptype IN ('a', 'i')))
				AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend AS dependency
					WHERE dependency.classid = object_class AND dependency.objid = object_id
					AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.deptype = 'e');
			WHEN 'pg_catalog.pg_proc'::regclass THEN
				SELECT pg_catalog.format('ALTER %s %I.%I(%s) OWNER TO %I', CASE routine.prokind
					WHEN 'p' THEN 'PROCEDURE' WHEN 'a' THEN 'AGGREGATE' ELSE 'FUNCTION' END,
					namespace.nspname, routine.proname, pg_catalog.pg_get_function_identity_arguments(routine.oid), @objects@)
				INTO statement FROM pg_catalog.pg_proc AS routine
				JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
				WHERE routine.oid = object_id AND routine.proowner = ANY(creators) AND @ordinary@
				AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend AS dependency
					WHERE dependency.classid = object_class AND dependency.objid = object_id
					AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.deptype = 'e');
			WHEN 'pg_catalog.pg_type'::regclass THEN
				SELECT pg_catalog.format('ALTER %s %I.%I OWNER TO %I', CASE WHEN type.typtype = 'd' THEN 'DOMAIN' ELSE 'TYPE' END,
					namespace.nspname, type.typname, @objects@)
				INTO statement FROM pg_catalog.pg_type AS type
				JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = type.typnamespace
				WHERE type.oid = object_id AND type.typowner = ANY(creators) AND type.typtype IN ('c', 'd', 'e', 'r')
				AND (type.typrelid = 0 OR EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid = type.typrelid AND relkind = 'c'))
				AND @ordinary@ AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend AS dependency
					WHERE dependency.classid = object_class AND dependency.objid = object_id
					AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.deptype = 'e');
			ELSE RETURN;
			END CASE;
			IF statement IS NOT NULL THEN EXECUTE statement; END IF;
		END;`)
	definition := "CREATE OR REPLACE FUNCTION " + normalize + "(object_class oid, object_id oid) RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS " + pgLiteral(body)
	if functionError := ownershipFunction(ctx, transaction, normalize+"(oid, oid)", definition, recipients); functionError != nil {
		return functionError
	}
	body = replacements.Replace(`DECLARE command record;
		BEGIN
			IF session_user NOT IN (@primary@, @operator@) THEN RETURN; END IF;
			FOR command IN SELECT DISTINCT classid, objid FROM pg_catalog.pg_event_trigger_ddl_commands()
				WHERE NOT in_extension AND classid IN ('pg_catalog.pg_namespace'::regclass,
				'pg_catalog.pg_class'::regclass, 'pg_catalog.pg_proc'::regclass, 'pg_catalog.pg_type'::regclass)
			LOOP PERFORM @normalize@(command.classid, command.objid); END LOOP;
		END;`)
	definition = "CREATE OR REPLACE FUNCTION " + guard + "() RETURNS event_trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS " + pgLiteral(body)
	if functionError := ownershipFunction(ctx, transaction, guard+"()", definition, recipients); functionError != nil {
		return functionError
	}
	var exists bool
	var marker string
	if queryError := transaction.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_event_trigger WHERE evtname = $1),
		COALESCE((SELECT pg_catalog.obj_description(oid, 'pg_event_trigger') FROM pg_catalog.pg_event_trigger WHERE evtname = $1), '')`, OwnershipGuard).Scan(&exists, &marker); queryError != nil {
		return SafeError(queryError)
	}
	if exists {
		var correct bool
		if queryError := transaction.QueryRow(ctx, `SELECT event.evtfoid = pg_catalog.to_regprocedure($2)
			AND owner.rolname = $3 FROM pg_catalog.pg_event_trigger AS event JOIN pg_catalog.pg_roles AS owner ON owner.oid = event.evtowner
			WHERE event.evtname = $1`, OwnershipGuard, guard+"()", config.Postgres.AdminUsername).Scan(&correct); queryError != nil {
			return SafeError(queryError)
		}
		if marker != OwnershipMarker || !correct {
			return IdentityMismatch
		}
	} else {
		if _, createError := transaction.Exec(ctx, "CREATE EVENT TRIGGER "+pgIdentifier(OwnershipGuard)+" ON ddl_command_end EXECUTE FUNCTION "+guard+"()"); createError != nil {
			return SafeError(createError)
		}
		if _, markerError := transaction.Exec(ctx, "COMMENT ON EVENT TRIGGER "+pgIdentifier(OwnershipGuard)+" IS "+pgLiteral(OwnershipMarker)); markerError != nil {
			return SafeError(markerError)
		}
	}
	if _, enableError := transaction.Exec(ctx, "ALTER EVENT TRIGGER "+pgIdentifier(OwnershipGuard)+" ENABLE ALWAYS"); enableError != nil {
		return SafeError(enableError)
	}
	objects, queryError := queryMetadataPolicyRows(ctx, transaction, `WITH creators AS (
		SELECT oid FROM pg_catalog.pg_roles WHERE rolname = ANY($1::text[]) AND pg_catalog.shobj_description(oid, 'pg_authid') = $2 || rolname
	) SELECT 'pg_catalog.pg_namespace'::regclass::oid, oid FROM pg_catalog.pg_namespace WHERE nspowner IN (SELECT oid FROM creators)
	UNION SELECT 'pg_catalog.pg_class'::regclass::oid, oid FROM pg_catalog.pg_class WHERE relowner IN (SELECT oid FROM creators)
	UNION SELECT 'pg_catalog.pg_proc'::regclass::oid, oid FROM pg_catalog.pg_proc WHERE proowner IN (SELECT oid FROM creators)
	UNION SELECT 'pg_catalog.pg_type'::regclass::oid, oid FROM pg_catalog.pg_type WHERE typowner IN (SELECT oid FROM creators)`, func(row pgx.CollectableRow) ([2]uint32, error) {
		var object [2]uint32
		err := row.Scan(&object[0], &object[1])
		return object, err
	}, []string{record.Name, roles.Owner}, databaseMarker(record)+":")
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, object := range objects {
		if _, ownershipError := transaction.Exec(ctx, "SELECT "+normalize+"($1::oid, $2::oid)", object[0], object[1]); ownershipError != nil {
			return SafeError(ownershipError)
		}
	}
	return nil
}

func ownershipFunction(ctx context.Context, transaction pgx.Tx, signature, definition, recipients string) error {
	var owner, marker string
	queryError := transaction.QueryRow(ctx, `SELECT owner.rolname, COALESCE(pg_catalog.obj_description(routine.oid, 'pg_proc'), '')
		FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_roles AS owner ON owner.oid = routine.proowner
		WHERE routine.oid = pg_catalog.to_regprocedure($1)`, signature).Scan(&owner, &marker)
	if queryError != nil && !errors.Is(queryError, pgx.ErrNoRows) {
		return SafeError(queryError)
	}
	if queryError == nil && (owner != config.Postgres.AdminUsername || marker != OwnershipMarker) {
		return IdentityMismatch
	}
	for _, command := range []string{definition, "COMMENT ON FUNCTION " + signature + " IS " + pgLiteral(OwnershipMarker), "REVOKE ALL ON FUNCTION " + signature + " FROM " + recipients} {
		if _, functionError := transaction.Exec(ctx, command); functionError != nil {
			return SafeError(functionError)
		}
	}
	return nil
}
