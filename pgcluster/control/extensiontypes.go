package control

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func protectExtensionTypes(ctx context.Context, transaction pgx.Tx, roles Roles, recipients string) error {
	rows, queryError := transaction.Query(ctx, `SELECT extension.extname, namespace.nspname, type_record.typname
		FROM pg_catalog.pg_extension AS extension JOIN pg_catalog.pg_depend AS dependency ON dependency.refobjid = extension.oid
		AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.classid = 'pg_catalog.pg_type'::regclass AND dependency.deptype = 'e'
		JOIN pg_catalog.pg_type AS type_record ON type_record.oid = dependency.objid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = type_record.typnamespace
		WHERE type_record.typtype <> 'p'
		AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_type AS element_type
			WHERE element_type.oid = type_record.typelem AND element_type.typarray = type_record.oid)
		ORDER BY extension.extname, type_record.oid`)
	if queryError != nil {
		return SafeError(queryError)
	}
	type typeGrant struct {
		Name        string
		Application bool
	}
	grants := []typeGrant{}
	for rows.Next() {
		var extension, schema, name string
		if scanError := rows.Scan(&extension, &schema, &name); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		grants = append(grants, typeGrant{Name: pgx.Identifier{schema, name}.Sanitize(), Application: extensionCatalog[extension].Supported && extensionCatalog[extension].NativeUsage == ApplicationExtension})
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, grant := range grants {
		if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON TYPE "+grant.Name+" FROM "+recipients+", "+pgIdentifier(roles.Owner)); revokeError != nil {
			return SafeError(revokeError)
		}
		if grant.Application {
			if _, grantError := transaction.Exec(ctx, "GRANT USAGE ON TYPE "+grant.Name+" TO "+pgIdentifier(roles.Owner)+", "+pgIdentifier(roles.ReadAccess)+", "+pgIdentifier(roles.WriteAccess)); grantError != nil {
				return SafeError(grantError)
			}
		}
	}
	return nil
}
