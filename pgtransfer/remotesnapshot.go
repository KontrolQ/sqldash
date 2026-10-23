package pgtransfer

import (
	"context"
	"strconv"
	"strings"

	"sqldash/models"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

type catalogQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (self *job) remoteSnapshot(source RemoteSource, endpoint *remoteEndpoint, operation func(*RemoteManifest, string, []string) error) error {
	connection, connectError := self.remoteConnection(source, endpoint)
	if connectError != nil {
		return connectError
	}
	defer closeSourceConnection(connection)
	var database, username string
	var version int
	verifyError := connection.QueryRow(self.context, "SELECT current_database(), session_user::text, current_setting('server_version_num')::integer").Scan(&database, &username, &version)
	if verifyError != nil || database != source.Database || username != source.Username {
		return self.safeError(RemoteConnectionFailed)
	}
	if version < MinimumRemoteServerVersion || version >= MaximumRemoteServerVersion {
		return RemoteVersionUnsupported
	}
	if identityError := self.remoteIdentity(source, connection, version); identityError != nil {
		return identityError
	}
	endpoint.serverVersion = version
	if version >= 90600 {
		if _, timeoutError := connection.Exec(self.context, "SELECT pg_catalog.set_config('idle_in_transaction_session_timeout', $1, false)", strconv.FormatInt(self.options.Timeout.Milliseconds(), 10)); timeoutError != nil {
			return self.safeError(RemoteConnectionFailed)
		}
	}
	return self.sourceTransaction(connection, func(transaction pgx.Tx) error {
		excluded, exclusionError := self.remoteControlSchemas(transaction, version)
		if exclusionError != nil {
			return exclusionError
		}
		extensions, extensionError := self.extensionManifest(transaction)
		if extensionError != nil {
			return extensionError
		}
		manifest := &RemoteManifest{ServerVersion: version, Database: database, Extensions: []RemoteExtension{}}
		for _, extension := range extensions {
			if extension.Name != "plpgsql" && extension.Name != "pg_stat_statements" {
				manifest.Extensions = append(manifest.Extensions, extension)
			}
		}
		schemas, schemasError := self.snapshotSchemas(transaction, excluded)
		if schemasError != nil {
			return schemasError
		}
		snapshot, snapshotError := self.exportSnapshot(transaction)
		if snapshotError != nil {
			return snapshotError
		}
		if operationError := operation(manifest, snapshot, schemas); operationError != nil {
			return operationError
		}
		return self.authorize()
	})
}

func (self *job) remoteIdentity(source RemoteSource, connection *pgx.Conn, version int) error {
	var administrator bool
	var marker string
	roleError := connection.QueryRow(self.context, `SELECT rolsuper,
		COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), '')
		FROM pg_catalog.pg_roles WHERE rolname = session_user`).Scan(&administrator, &marker)
	if roleError != nil {
		return self.safeError(RemoteMetadataInvalid)
	}
	if internalRemoteUsername(source.Username) && strings.HasPrefix(marker, "sqldash:") && strings.HasSuffix(marker, ":"+source.Username) {
		return RemoteSourceDenied
	}
	if version < 180000 || !administrator {
		return nil
	}
	var remoteIdentifier string
	if identityError := connection.QueryRow(self.context, "SELECT system_identifier::text FROM pg_catalog.pg_control_system()").Scan(&remoteIdentifier); identityError != nil {
		return self.safeError(RemoteMetadataInvalid)
	}
	identifier, identityError := control.ClusterIdentity(self.context)
	if identityError != nil || identifier == "" {
		return RemoteSourceDenied
	}
	if remoteIdentifier == identifier {
		return RemoteSourceDenied
	}
	return nil
}

func (self *job) remoteControlSchemas(transaction pgx.Tx, version int) ([]string, error) {
	if version < 90300 {
		return []string{}, nil
	}
	rows, queryError := transaction.Query(self.context, `SELECT trigger_record.evtname,
		COALESCE(pg_catalog.obj_description(trigger_record.oid, 'pg_event_trigger'), ''),
		namespace.nspname, routine.proname, owner.rolsuper,
		EXISTS(SELECT 1 FROM pg_catalog.pg_depend AS dependency
		WHERE dependency.classid = 'pg_catalog.pg_event_trigger'::regclass
		AND dependency.objid = trigger_record.oid
		AND dependency.refclassid = 'pg_catalog.pg_extension'::regclass AND dependency.deptype = 'e')
		FROM pg_catalog.pg_event_trigger AS trigger_record
		JOIN pg_catalog.pg_proc AS routine ON routine.oid = trigger_record.evtfoid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		JOIN pg_catalog.pg_roles AS owner ON owner.oid = routine.proowner`)
	if queryError != nil {
		return nil, self.safeError(RemoteMetadataInvalid)
	}
	defer rows.Close()
	excluded := []string{}
	for rows.Next() {
		var name, marker, schema, function string
		var administrator, extensionOwned bool
		if scanError := rows.Scan(&name, &marker, &schema, &function, &administrator, &extensionOwned); scanError != nil {
			return nil, self.safeError(RemoteMetadataInvalid)
		}
		if extensionOwned {
			continue
		}
		creationGuard := name == control.ExtensionGuard && marker == control.ExtensionGuardMarker && function == control.ExtensionGuardFunction
		dropGuard := name == control.ExtensionDropGuard && marker == control.ExtensionDropGuardMarker && function == control.ExtensionDropGuardFunction
		if (!creationGuard && !dropGuard) || schema != control.ControlSchema || !administrator {
			return nil, RemoteObjectsUnsupported
		}
		excluded = []string{control.ControlSchema, control.ExtensionsSchema}
	}
	if queryError := rows.Err(); queryError != nil {
		return nil, self.safeError(RemoteMetadataInvalid)
	}
	return excluded, nil
}

func (self *job) extensionManifest(catalog catalogQuerier) ([]RemoteExtension, error) {
	rows, queryError := catalog.Query(self.context, `SELECT extension.extname, extension.extversion, namespace.nspname
		FROM pg_catalog.pg_extension AS extension
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = extension.extnamespace
		ORDER BY extension.extname`)
	if queryError != nil {
		return nil, self.safeError(RemoteMetadataInvalid)
	}
	defer rows.Close()
	extensions := []RemoteExtension{}
	for rows.Next() {
		var extension RemoteExtension
		if scanError := rows.Scan(&extension.Name, &extension.Version, &extension.Schema); scanError != nil {
			return nil, self.safeError(RemoteMetadataInvalid)
		}
		if len(extensions) >= MaximumSourceExtensions || !validRemoteIdentifier(extension.Name) || !validRemoteIdentifier(extension.Schema) || !validRemoteText(extension.Version, 128) {
			return nil, RemoteMetadataInvalid
		}
		extensions = append(extensions, extension)
	}
	if queryError := rows.Err(); queryError != nil {
		return nil, self.safeError(RemoteMetadataInvalid)
	}
	return extensions, nil
}

func (self *job) remoteRequirements(target *models.Database, manifest *RemoteManifest) error {
	if len(manifest.Extensions) == 0 {
		return nil
	}
	return self.withLogin(target, control.ReadOnly, func(credential *control.TransferCredential) error {
		connection, connectError := self.privateConnection(target, credential)
		if connectError != nil {
			return connectError
		}
		defer closeSourceConnection(connection)
		installed, installedError := self.extensionManifest(connection)
		if installedError != nil {
			return installedError
		}
		byName := make(map[string]RemoteExtension, len(installed))
		for _, extension := range installed {
			byName[extension.Name] = extension
		}
		missing := []RemoteExtension{}
		for _, required := range manifest.Extensions {
			extension, exists := byName[required.Name]
			if !exists || extension.Version != required.Version || extension.Schema != required.Schema {
				missing = append(missing, required)
			}
		}
		if len(missing) != 0 {
			return &RemoteRequirementsError{Requirements: missing}
		}
		return nil
	})
}
