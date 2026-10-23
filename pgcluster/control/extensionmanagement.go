package control

import (
	"context"
	"errors"
	"strings"

	"sqldash/config"
	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

var ExtensionUnavailable = errors.New("the requested extension version or runtime prerequisite is unavailable")

func ManageExtension(ctx context.Context, record *models.Database, name string, install bool) error {
	return manageExtension(ctx, record, name, "", "", install)
}

func ManageExtensionInSchema(ctx context.Context, record *models.Database, name string, install bool, schema string) error {
	return manageExtension(ctx, record, name, "", schema, install)
}

func ManageExtensionVersion(ctx context.Context, record *models.Database, name, version, schema string) error {
	if version == "" {
		return ExtensionUnavailable
	}
	return manageExtension(ctx, record, name, version, schema, true)
}

func manageExtension(ctx context.Context, record *models.Database, name, version, schema string, install bool) error {
	if !supportedExtension(name) {
		return ExtensionUnavailable
	}
	if record == nil || record.State != models.DatabaseReady {
		return InvalidDatabase
	}
	return changeExtension(ctx, record, name, version, schema, install)
}

func changeExtension(ctx context.Context, record *models.Database, name, version, schema string, install bool) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, record.Name)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		return lockError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if install {
		if installError := installExtension(ctx, transaction, record, roles, name, version, schema); installError != nil {
			return installError
		}
	} else {
		if _, dropError := transaction.Exec(ctx, "DROP EXTENSION IF EXISTS "+pgIdentifier(name)+" RESTRICT"); dropError != nil {
			return SafeError(dropError)
		}
	}
	if policyError := applyPolicy(ctx, transaction, record, roles); policyError != nil {
		return policyError
	}
	return SafeError(transaction.Commit(ctx))
}

func installExtension(ctx context.Context, transaction pgx.Tx, record *models.Database, roles Roles, name, version, schema string) error {
	if !supportedExtension(name) {
		return ExtensionUnavailable
	}
	var availableVersion string
	var fixedSchema *string
	var requirements []string
	queryError := transaction.QueryRow(ctx, `SELECT version.version, version.schema, COALESCE(version.requires::text[], ARRAY[]::text[])
		FROM pg_catalog.pg_available_extension_versions AS version JOIN pg_catalog.pg_available_extensions AS available ON available.name = version.name
		WHERE version.name = $1 AND version.version = CASE WHEN $2 = '' THEN available.default_version ELSE $2 END`, name, version).Scan(&availableVersion, &fixedSchema, &requirements)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return ExtensionUnavailable
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	if fixedSchema != nil && *fixedSchema != "" {
		if schema != "" && schema != *fixedSchema {
			return ExtensionUnavailable
		}
		schema = *fixedSchema
	}
	if schema == "" {
		schema = ExtensionsSchema
	}
	if !validExtensionSchema(schema) {
		return InvalidDatabase
	}
	var installedVersion, installedSchema, installedOwner string
	queryError = transaction.QueryRow(ctx, `SELECT extension.extversion, namespace.nspname, owner.rolname
		FROM pg_catalog.pg_extension AS extension JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = extension.extnamespace
		JOIN pg_catalog.pg_roles AS owner ON owner.oid = extension.extowner WHERE extension.extname = $1`, name).Scan(&installedVersion, &installedSchema, &installedOwner)
	if queryError == nil {
		if installedOwner != config.Postgres.AdminUsername || installedVersion != availableVersion || installedSchema != schema {
			return IdentityMismatch
		}
		return nil
	}
	if !errors.Is(queryError, pgx.ErrNoRows) {
		return SafeError(queryError)
	}
	if prerequisiteError := extensionPrerequisites(ctx, transaction, record, name); prerequisiteError != nil {
		return prerequisiteError
	}
	for _, requirement := range requirements {
		if requirement == "plpgsql" {
			continue
		}
		var exists bool
		if queryError := transaction.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_extension WHERE extname = $1)", requirement).Scan(&exists); queryError != nil {
			return SafeError(queryError)
		}
		if !exists {
			return &ExtensionDependencyError{Name: requirement}
		}
	}
	if schemaError := extensionNamespace(ctx, transaction, roles, schema); schemaError != nil {
		return schemaError
	}
	command := "CREATE EXTENSION " + pgIdentifier(name) + " WITH SCHEMA " + pgIdentifier(schema) + " VERSION " + pgLiteral(availableVersion)
	_, createError := transaction.Exec(ctx, command)
	return SafeError(createError)
}

func validExtensionSchema(schema string) bool {
	return validIdentifier(schema) && !strings.HasPrefix(strings.ToLower(schema), "pg_") && schema != "information_schema" && schema != ControlSchema
}

func extensionNamespace(ctx context.Context, transaction pgx.Tx, roles Roles, schema string) error {
	var owner string
	queryError := transaction.QueryRow(ctx, `SELECT COALESCE((SELECT owner.rolname FROM pg_catalog.pg_namespace AS namespace
		JOIN pg_catalog.pg_roles AS owner ON owner.oid = namespace.nspowner WHERE namespace.nspname = $1), '')`, schema).Scan(&owner)
	if queryError != nil {
		return SafeError(queryError)
	}
	if owner != "" {
		if owner != config.Postgres.AdminUsername && owner != roles.Owner {
			return IdentityMismatch
		}
		return nil
	}
	_, createError := transaction.Exec(ctx, "CREATE SCHEMA "+pgIdentifier(schema)+" AUTHORIZATION "+pgIdentifier(config.Postgres.AdminUsername))
	return SafeError(createError)
}

type extensionQuerySource interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func extensionPrerequisites(ctx context.Context, transaction extensionQuerySource, record *models.Database, name string) error {
	preloads := map[string]string{
		"pg_stat_statements": "pg_stat_statements", "pg_cron": "pg_cron", "pg_squeeze": "pg_squeeze", "pg_stat_kcache": "pg_stat_kcache",
		"pg_qualstats": "pg_qualstats", "pg_hint_plan": "pg_hint_plan", "pgaudit": "pgaudit", "anon": "anon", "timescaledb": "timescaledb",
	}
	if preload := preloads[name]; preload != "" {
		var loaded string
		if queryError := transaction.QueryRow(ctx, "SELECT current_setting('shared_preload_libraries')").Scan(&loaded); queryError != nil {
			return SafeError(queryError)
		}
		found := false
		for _, library := range strings.Split(loaded, ",") {
			if strings.Trim(strings.TrimSpace(library), "\"") == preload {
				found = true
			}
		}
		if !found {
			return ExtensionUnavailable
		}
	}
	if name == "pg_cron" {
		var database string
		if queryError := transaction.QueryRow(ctx, "SELECT COALESCE(current_setting('cron.database_name', true), '')").Scan(&database); queryError != nil {
			return SafeError(queryError)
		}
		if database != record.Name {
			return ExtensionUnavailable
		}
	}
	return nil
}
