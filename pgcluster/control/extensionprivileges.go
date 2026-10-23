package control

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func protectExtensionRoutines(ctx context.Context, transaction pgx.Tx, roles Roles, recipients string) error {
	rows, queryError := transaction.Query(ctx, `SELECT extension.extname, namespace.nspname, routine.proname,
		pg_catalog.pg_get_function_identity_arguments(routine.oid), routine.prokind::text
		FROM pg_catalog.pg_extension AS extension JOIN pg_catalog.pg_depend AS dependency ON dependency.refobjid = extension.oid
		AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.classid = 'pg_catalog.pg_proc'::regclass AND dependency.deptype = 'e'
		JOIN pg_catalog.pg_proc AS routine ON routine.oid = dependency.objid JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE extension.extname <> 'plpgsql' ORDER BY extension.extname, routine.oid`)
	if queryError != nil {
		return SafeError(queryError)
	}
	type routineGrant struct {
		Name        string
		Application bool
	}
	grants := []routineGrant{}
	for rows.Next() {
		var extension, schema, name, arguments, kind string
		if scanError := rows.Scan(&extension, &schema, &name, &arguments, &kind); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		identifier := pgx.Identifier{schema, name}.Sanitize() + "(" + arguments + ")"
		grant := routineGrant{Name: identifier, Application: extensionCatalog[extension].Supported && extensionCatalog[extension].NativeUsage == ApplicationExtension}
		grants = append(grants, grant)
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, grant := range grants {
		if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON ROUTINE "+grant.Name+" FROM "+recipients+", "+pgIdentifier(roles.Owner)); revokeError != nil {
			return SafeError(revokeError)
		}
		if grant.Application {
			if _, grantError := transaction.Exec(ctx, "GRANT EXECUTE ON ROUTINE "+grant.Name+" TO "+pgIdentifier(roles.Owner)+", "+pgIdentifier(roles.WriteAccess)); grantError != nil {
				return SafeError(grantError)
			}
		}
	}
	return nil
}
