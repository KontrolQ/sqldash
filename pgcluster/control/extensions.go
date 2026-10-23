package control

import (
	"context"
	"sort"

	"sqldash/config"
	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

const (
	ControlSchema          = "sqd_control"
	ExtensionGuard         = "sqd_extension_guard"
	ExtensionGuardFunction = "guard_extensions"
	ExtensionGuardMarker   = "sqldash:extension-guard"
	ExtensionsSchema       = "sqd_extensions"
)

type ExtensionRecord struct {
	Name             string
	DefaultVersion   string
	InstalledVersion string
	Comment          string
	Supported        bool
	Protected        bool
	Available        bool
	Enabled          bool
	SourceKey        string
	Type             string
	NativeUsage      string
	Reason           string
	Schema           string
	Label            string
}

func guardExtensions(ctx context.Context, transaction pgx.Tx, recipients string) error {
	var schemaOwner string
	queryError := transaction.QueryRow(ctx, `SELECT COALESCE((SELECT owner.rolname FROM pg_catalog.pg_namespace AS namespace
		JOIN pg_catalog.pg_roles AS owner ON owner.oid = namespace.nspowner WHERE namespace.nspname = $1), '')`, ControlSchema).Scan(&schemaOwner)
	if queryError != nil {
		return SafeError(queryError)
	}
	if schemaOwner == "" {
		if _, createError := transaction.Exec(ctx, "CREATE SCHEMA "+pgIdentifier(ControlSchema)+" AUTHORIZATION "+pgIdentifier(config.Postgres.AdminUsername)); createError != nil {
			return SafeError(createError)
		}
	} else if schemaOwner != config.Postgres.AdminUsername {
		return IdentityMismatch
	}
	var existingMarker string
	queryError = transaction.QueryRow(ctx, `SELECT COALESCE((SELECT pg_catalog.obj_description(oid, 'pg_event_trigger')
		FROM pg_catalog.pg_event_trigger WHERE evtname = $1), '')`, ExtensionGuard).Scan(&existingMarker)
	if queryError != nil {
		return SafeError(queryError)
	}
	var exists bool
	queryError = transaction.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_event_trigger WHERE evtname = $1)", ExtensionGuard).Scan(&exists)
	if queryError != nil {
		return SafeError(queryError)
	}
	if exists && existingMarker != ExtensionGuardMarker {
		return IdentityMismatch
	}
	function := pgx.Identifier{ControlSchema, ExtensionGuardFunction}.Sanitize()
	definition := "CREATE OR REPLACE FUNCTION " + function + "() RETURNS event_trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $sqd$ BEGIN IF session_user <> " + pgLiteral(config.Postgres.AdminUsername) + " THEN RAISE EXCEPTION 'extension management is not available in SQL sessions' USING ERRCODE = '42501'; END IF; END; $sqd$"
	if _, createError := transaction.Exec(ctx, definition); createError != nil {
		return SafeError(createError)
	}
	if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON SCHEMA "+pgIdentifier(ControlSchema)+" FROM "+recipients); revokeError != nil {
		return SafeError(revokeError)
	}
	if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON FUNCTION "+function+"() FROM "+recipients); revokeError != nil {
		return SafeError(revokeError)
	}
	if !exists {
		command := "CREATE EVENT TRIGGER " + pgIdentifier(ExtensionGuard) + " ON ddl_command_start WHEN TAG IN ('CREATE EXTENSION', 'ALTER EXTENSION', 'DROP EXTENSION') EXECUTE FUNCTION " + function + "()"
		if _, createError := transaction.Exec(ctx, command); createError != nil {
			return SafeError(createError)
		}
		if _, commentError := transaction.Exec(ctx, "COMMENT ON EVENT TRIGGER "+pgIdentifier(ExtensionGuard)+" IS "+pgLiteral(ExtensionGuardMarker)); commentError != nil {
			return SafeError(commentError)
		}
	} else {
		var correctFunction bool
		queryError := transaction.QueryRow(ctx, `SELECT trigger_record.evtfoid = routine.oid FROM pg_catalog.pg_event_trigger AS trigger_record
			JOIN pg_catalog.pg_proc AS routine ON routine.proname = $2
			JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace AND namespace.nspname = $3
			WHERE trigger_record.evtname = $1`, ExtensionGuard, ExtensionGuardFunction, ControlSchema).Scan(&correctFunction)
		if queryError != nil {
			return SafeError(queryError)
		}
		if !correctFunction {
			return IdentityMismatch
		}
	}
	if _, enableError := transaction.Exec(ctx, "ALTER EVENT TRIGGER "+pgIdentifier(ExtensionGuard)+" ENABLE ALWAYS"); enableError != nil {
		return SafeError(enableError)
	}
	return nil
}

func validateExtensions(ctx context.Context, connection *pgx.Conn) error {
	rows, queryError := connection.Query(ctx, "SELECT extname FROM pg_catalog.pg_extension")
	if queryError != nil {
		return SafeError(queryError)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if scanError := rows.Scan(&name); scanError != nil {
			return SafeError(scanError)
		}
		if name != "plpgsql" && name != "pg_stat_statements" && !supportedExtension(name) {
			return RoleMismatch
		}
	}
	return SafeError(rows.Err())
}

func SupportedExtensions() []string {
	names := make([]string, 0, len(extensionCatalog))
	for name, descriptor := range extensionCatalog {
		if descriptor.Supported {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func Extensions(ctx context.Context, record *models.Database) ([]ExtensionRecord, error) {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return nil, roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, record.Name)
	if connectError != nil {
		return nil, connectError
	}
	defer closeConnection(connection)
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return nil, verifyError
	}
	rows, queryError := connection.Query(ctx, `SELECT available.name, COALESCE(available.default_version, ''), COALESCE(available.installed_version, ''), COALESCE(available.comment, ''), COALESCE(namespace.nspname, '')
		FROM pg_catalog.pg_available_extensions AS available LEFT JOIN pg_catalog.pg_extension AS installed ON installed.extname = available.name
		LEFT JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = installed.extnamespace WHERE available.name <> ALL($1::text[]) AND available.name !~ '_fdw$'
		AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend AS dependency WHERE dependency.refclassid = 'pg_catalog.pg_extension'::regclass
		AND dependency.refobjid = installed.oid AND dependency.classid = 'pg_catalog.pg_foreign_data_wrapper'::regclass AND dependency.deptype = 'e') ORDER BY available.name`, []string{"postgres_fdw", "file_fdw", "mysql_fdw", "mongo_fdw", "sqlite_fdw"})
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	inventory := map[string]ExtensionRecord{}
	for rows.Next() {
		var extension ExtensionRecord
		if scanError := rows.Scan(&extension.Name, &extension.DefaultVersion, &extension.InstalledVersion, &extension.Comment, &extension.Schema); scanError != nil {
			rows.Close()
			return nil, SafeError(scanError)
		}
		extension.Available = true
		extension.Enabled = extension.InstalledVersion != ""
		extension.Protected = extension.Name == "plpgsql"
		inventory[extension.Name] = extension
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	extensions := make([]ExtensionRecord, 0, len(inventory))
	for name, extension := range inventory {
		descriptor := extensionCatalog[name]
		if descriptor.Supported {
			if prerequisiteError := extensionPrerequisites(ctx, connection, record, name); prerequisiteError != nil {
				if prerequisiteError == ExtensionUnavailable {
					continue
				}
				return nil, prerequisiteError
			}
		}
		extension.Supported = descriptor.Supported
		extension.SourceKey = descriptor.SourceKey
		extension.NativeUsage = descriptor.NativeUsage
		extension.Reason = descriptor.Reason
		extension.Type = "extension"
		extension.Label = name
		switch name {
		case "vector":
			extension.Label = "pgvector"
		case "anon":
			extension.Label = "pg_anonymize"
		case "semver":
			extension.Label = "pg_semver"
		case "roaringbitmap":
			extension.Label = "pg_roaringbitmap"
		case "hll":
			extension.Label = "PostgreSQL HLL"
		}
		if name == "plpgsql" {
			extension.SourceKey = ContribSource
		}
		extensions = append(extensions, extension)
	}
	sort.Slice(extensions, func(first, second int) bool { return extensions[first].Name < extensions[second].Name })
	return extensions, nil
}

func ensureExtensionsSchema(ctx context.Context, transaction pgx.Tx) error {
	var owner string
	queryError := transaction.QueryRow(ctx, `SELECT COALESCE((SELECT role.rolname FROM pg_catalog.pg_namespace AS namespace
		JOIN pg_catalog.pg_roles AS role ON role.oid = namespace.nspowner WHERE namespace.nspname = $1), '')`, ExtensionsSchema).Scan(&owner)
	if queryError != nil {
		return SafeError(queryError)
	}
	if owner != "" {
		if owner != config.Postgres.AdminUsername {
			return IdentityMismatch
		}
		return nil
	}
	_, createError := transaction.Exec(ctx, "CREATE SCHEMA "+pgIdentifier(ExtensionsSchema)+" AUTHORIZATION "+pgIdentifier(config.Postgres.AdminUsername))
	return SafeError(createError)
}

func protectExtensions(ctx context.Context, transaction pgx.Tx, roles Roles, recipients string) error {
	var exists bool
	if queryError := transaction.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname = $1)", ExtensionsSchema).Scan(&exists); queryError != nil {
		return SafeError(queryError)
	}
	if !exists {
		return nil
	}
	identifier := pgIdentifier(ExtensionsSchema)
	commands := []string{
		"REVOKE ALL ON SCHEMA " + identifier + " FROM " + recipients + ", " + pgIdentifier(roles.Owner),
		"GRANT USAGE ON SCHEMA " + identifier + " TO " + pgIdentifier(roles.Owner) + ", " + pgIdentifier(roles.ReadAccess) + ", " + pgIdentifier(roles.WriteAccess),
		"REVOKE ALL ON ALL ROUTINES IN SCHEMA " + identifier + " FROM " + recipients + ", " + pgIdentifier(roles.Owner),
	}
	for _, command := range commands {
		if _, privilegeError := transaction.Exec(ctx, command); privilegeError != nil {
			return SafeError(privilegeError)
		}
	}
	return nil
}
