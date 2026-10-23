package control

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func protectLargeObjects(ctx context.Context, transaction pgx.Tx, roles Roles, recipients string) error {
	rows, queryError := transaction.Query(ctx, `SELECT routine.proname, pg_catalog.pg_get_function_identity_arguments(routine.oid)
		FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE namespace.nspname = 'pg_catalog' AND routine.proname IN ('lo_create', 'lo_creat', 'lo_from_bytea')`)
	if queryError != nil {
		return SafeError(queryError)
	}
	functions := []string{}
	for rows.Next() {
		var name, arguments string
		if scanError := rows.Scan(&name, &arguments); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		functions = append(functions, pgx.Identifier{"pg_catalog", name}.Sanitize()+"("+arguments+")")
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, function := range functions {
		if _, revokeError := transaction.Exec(ctx, "REVOKE EXECUTE ON FUNCTION "+function+" FROM "+recipients); revokeError != nil {
			return SafeError(revokeError)
		}
		if _, grantError := transaction.Exec(ctx, "GRANT EXECUTE ON FUNCTION "+function+" TO "+pgIdentifier(roles.Owner)); grantError != nil {
			return SafeError(grantError)
		}
	}
	rows, queryError = transaction.Query(ctx, "SELECT oid FROM pg_catalog.pg_largeobject_metadata ORDER BY oid")
	if queryError != nil {
		return SafeError(queryError)
	}
	objects := []uint32{}
	for rows.Next() {
		var identifier uint32
		if scanError := rows.Scan(&identifier); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		objects = append(objects, identifier)
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, identifier := range objects {
		object := strconv.FormatUint(uint64(identifier), 10)
		commands := []string{
			"REVOKE ALL ON LARGE OBJECT " + object + " FROM " + recipients,
			"GRANT SELECT ON LARGE OBJECT " + object + " TO " + pgIdentifier(roles.ReadAccess),
			"GRANT SELECT, UPDATE ON LARGE OBJECT " + object + " TO " + pgIdentifier(roles.WriteAccess),
		}
		for _, command := range commands {
			if _, grantError := transaction.Exec(ctx, command); grantError != nil {
				return SafeError(grantError)
			}
		}
	}
	return nil
}
