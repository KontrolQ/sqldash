package pgtransfer

import (
	"context"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"sqldash/models"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

func (self *job) withSnapshot(record *models.Database, credential *control.TransferCredential, operation func(string, []string) error) error {
	connection, connectError := self.privateConnection(record, credential)
	if connectError != nil {
		return connectError
	}
	defer closeSourceConnection(connection)
	return self.sourceTransaction(connection, func(transaction pgx.Tx) error {
		schemas, schemasError := self.snapshotSchemas(transaction, []string{control.ControlSchema, control.ExtensionsSchema})
		if schemasError != nil {
			return schemasError
		}
		snapshot, snapshotError := self.exportSnapshot(transaction)
		if snapshotError != nil {
			return snapshotError
		}
		return operation(snapshot, schemas)
	})
}

func (self *job) privateConnection(record *models.Database, credential *control.TransferCredential) (*pgx.Conn, error) {
	configuration, configurationError := metadataConfiguration()
	if configurationError != nil {
		return nil, SnapshotFailed
	}
	host, portText, addressError := net.SplitHostPort(credential.Address)
	port, portError := strconv.ParseUint(portText, 10, 16)
	if addressError != nil || portError != nil {
		return nil, InvalidAddress
	}
	configuration.Host = host
	configuration.Port = uint16(port)
	configuration.Database = credential.Database
	configuration.User = credential.Username
	configuration.Password = credential.Password
	configuration.TLSConfig = nil
	configuration.Fallbacks = nil
	configuration.Tracer = nil
	configuration.MaxProtocolMessageBodyLen = MaximumMetadataMessageBytes
	configuration.RuntimeParams = self.snapshotSettings()
	connection, connectError := pgx.ConnectConfig(self.context, configuration)
	if connectError != nil {
		return nil, self.safeError(SnapshotFailed)
	}
	var database, username string
	var databaseOID uint32
	var version int
	var privileged bool
	verifyError := connection.QueryRow(self.context, `SELECT current_database(), session_user::text,
		(SELECT oid FROM pg_catalog.pg_database WHERE datname = current_database()),
		current_setting('server_version_num')::integer,
		(SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls
		FROM pg_catalog.pg_roles WHERE rolname = session_user)`).Scan(&database, &username, &databaseOID, &version, &privileged)
	if verifyError != nil || database != credential.Database || username != credential.Username || databaseOID != record.BackendOID || privileged || version < 180000 || version >= 190000 {
		closeSourceConnection(connection)
		return nil, self.safeError(SnapshotFailed)
	}
	return connection, nil
}

func (self *job) snapshotSettings() map[string]string {
	return map[string]string{
		"application_name":                    "sqldash-transfer-snapshot",
		"client_encoding":                     "UTF8",
		"standard_conforming_strings":         "on",
		"default_transaction_read_only":       "on",
		"statement_timeout":                   "30000",
		"lock_timeout":                        "5000",
		"search_path":                         "pg_catalog",
		"idle_in_transaction_session_timeout": strconv.FormatInt(self.options.Timeout.Milliseconds(), 10),
	}
}

func (self *job) sourceTransaction(connection *pgx.Conn, operation func(pgx.Tx) error) error {
	transaction, beginError := connection.BeginTx(self.context, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if beginError != nil {
		return self.safeError(SnapshotFailed)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), CleanupTimeout)
		defer cancel()
		transaction.Rollback(ctx)
	}()
	return operation(transaction)
}

func (self *job) snapshotSchemas(transaction pgx.Tx, excluded []string) ([]string, error) {
	if excluded == nil {
		excluded = []string{}
	}
	rows, queryError := transaction.Query(self.context, `SELECT nspname FROM pg_catalog.pg_namespace
		WHERE nspname <> 'information_schema' AND nspname NOT LIKE 'pg\_%' ESCAPE '\'
		AND nspname <> ALL($1::text[]) ORDER BY nspname`, excluded)
	if queryError != nil {
		return nil, self.safeError(SnapshotFailed)
	}
	schemas := []string{}
	for rows.Next() {
		var schema string
		if scanError := rows.Scan(&schema); scanError != nil {
			rows.Close()
			return nil, self.safeError(SnapshotFailed)
		}
		if schema == "" || len(schema) > 63 || !utf8.ValidString(schema) || strings.ContainsRune(schema, 0) {
			rows.Close()
			return nil, RemoteMetadataInvalid
		}
		if len(schemas) >= MaximumSourceSchemas {
			rows.Close()
			return nil, SizeExceeded
		}
		schemas = append(schemas, pgx.Identifier{schema}.Sanitize())
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil || len(schemas) == 0 {
		return nil, self.safeError(SnapshotFailed)
	}
	return schemas, nil
}

func (self *job) exportSnapshot(transaction pgx.Tx) (string, error) {
	var snapshot string
	if snapshotError := transaction.QueryRow(self.context, "SELECT pg_catalog.pg_export_snapshot()").Scan(&snapshot); snapshotError != nil || !validSnapshot(snapshot) {
		return "", self.safeError(SnapshotFailed)
	}
	return snapshot, nil
}

func closeSourceConnection(connection *pgx.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), CleanupTimeout)
	defer cancel()
	connection.Close(ctx)
}

func validSnapshot(snapshot string) bool {
	return snapshot != "" && len(snapshot) <= 128 && strings.IndexFunc(snapshot, func(character rune) bool {
		return character != '-' && (character < '0' || character > '9') && (character < 'A' || character > 'Z') && (character < 'a' || character > 'z')
	}) == -1
}
