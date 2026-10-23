package control

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func protectExtensionRelations(ctx context.Context, transaction pgx.Tx, roles Roles, recipients string) error {
	rows, queryError := transaction.Query(ctx, `SELECT extension.extname, namespace.nspname, relation.relname, relation.relkind::text
		FROM pg_catalog.pg_extension AS extension JOIN pg_catalog.pg_depend AS dependency ON dependency.refobjid = extension.oid
		AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.classid = 'pg_catalog.pg_class'::regclass AND dependency.deptype = 'e'
		JOIN pg_catalog.pg_class AS relation ON relation.oid = dependency.objid JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE relation.relkind IN ('r', 'p', 'v', 'm', 'f', 'S') ORDER BY extension.extname, relation.oid`)
	if queryError != nil {
		return SafeError(queryError)
	}
	type relationGrant struct {
		Name        string
		Sequence    bool
		Application bool
	}
	grants := []relationGrant{}
	for rows.Next() {
		var extension, schema, name, kind string
		if scanError := rows.Scan(&extension, &schema, &name, &kind); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		grants = append(grants, relationGrant{Name: pgx.Identifier{schema, name}.Sanitize(), Sequence: kind == "S", Application: extensionCatalog[extension].Supported && extensionCatalog[extension].NativeUsage == ApplicationExtension})
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, grant := range grants {
		kind := "TABLE "
		if grant.Sequence {
			kind = "SEQUENCE "
		}
		if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON "+kind+grant.Name+" FROM "+recipients+", "+pgIdentifier(roles.Owner)); revokeError != nil {
			return SafeError(revokeError)
		}
		if grant.Application {
			if _, grantError := transaction.Exec(ctx, "GRANT SELECT ON "+kind+grant.Name+" TO "+pgIdentifier(roles.Owner)+", "+pgIdentifier(roles.ReadAccess)+", "+pgIdentifier(roles.WriteAccess)); grantError != nil {
				return SafeError(grantError)
			}
		}
	}
	return nil
}
