package control

import (
	"context"
	"errors"
	"net"
	"strconv"

	"sqldash/config"
	"sqldash/models"
	"sqldash/pgcluster/runtime"
	"sqldash/secretstore"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
)

func connectionConfiguration(database, username, password string) (*pgx.ConnConfig, error) {
	if !config.Postgres.Enabled {
		return nil, Unavailable
	}
	host, portText, addressError := net.SplitHostPort(config.Postgres.Address)
	port, portError := strconv.ParseUint(portText, 10, 16)
	address := net.ParseIP(host)
	if addressError != nil || portError != nil || port == 0 || address == nil || !address.IsLoopback() {
		return nil, Unavailable
	}
	configuration, parseError := pgx.ParseConfig("host=127.0.0.1 port=55432 sslmode=disable connect_timeout=5 target_session_attrs=any require_auth=scram-sha-256")
	if parseError != nil {
		return nil, Unavailable
	}
	configuration.Host = host
	configuration.Port = uint16(port)
	configuration.Database = database
	configuration.User = username
	configuration.Password = password
	configuration.TLSConfig = nil
	configuration.Fallbacks = nil
	configuration.Tracer = nil
	configuration.RequireAuth = "scram-sha-256"
	// Environment target-session hooks can issue SQL before the caller receives the session.
	configuration.ValidateConnect = nil
	configuration.AfterConnect = nil
	configuration.ConnectTimeout = ConnectionTimeout
	configuration.RuntimeParams = map[string]string{
		"application_name":                    "sqldash",
		"statement_timeout":                   "30000",
		"lock_timeout":                        "5000",
		"idle_in_transaction_session_timeout": "30000",
		"standard_conforming_strings":         "on",
		"search_path":                         "pg_catalog, public",
		"TimeZone":                            "UTC",
	}
	return configuration, nil
}

func privateAdminConnect(ctx context.Context, database string) (*pgx.Conn, string, error) {
	password, passwordError := runtime.New().AdminPassword()
	if passwordError != nil {
		return nil, "", Unavailable
	}
	configuration, configurationError := connectionConfiguration(database, config.Postgres.AdminUsername, password)
	if configurationError != nil {
		return nil, "", configurationError
	}
	configuration.RuntimeParams["log_statement"] = "none"
	configuration.RuntimeParams["log_min_error_statement"] = "panic"
	configuration.RuntimeParams["log_parameter_max_length"] = "0"
	configuration.RuntimeParams["log_parameter_max_length_on_error"] = "0"
	connection, connectError := authenticatedConnect(ctx, configuration)
	if connectError != nil {
		return nil, "", connectError
	}
	identifier, verifyError := verifyCluster(ctx, connection)
	if verifyError != nil {
		closeConnection(connection)
		return nil, "", verifyError
	}
	return connection, identifier, nil
}

func verifyCluster(ctx context.Context, connection *pgx.Conn) (string, error) {
	var version int
	var administrator bool
	var identifier, username, database string
	queryError := connection.QueryRow(ctx, `SELECT current_setting('server_version_num')::integer,
		(SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname = session_user),
		(SELECT system_identifier::text FROM pg_catalog.pg_control_system()), session_user::text, current_database()::text`).Scan(&version, &administrator, &identifier, &username, &database)
	if queryError != nil {
		return "", SafeError(queryError)
	}
	if version < 180000 || version >= 190000 {
		return "", VersionMismatch
	}
	if !administrator || identifier == "" || username != config.Postgres.AdminUsername || database != connection.Config().Database {
		return "", IdentityMismatch
	}
	expected, identityError := runtime.New().SystemIdentifier()
	if identityError != nil || expected != "" && expected != identifier {
		return "", IdentityMismatch
	}
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return "", Unavailable
	}
	sealedIdentifier, sealError := store.GetOrCreate(ClusterNamespace, ClusterIdentitySecret, func() ([]byte, error) {
		return []byte(identifier), nil
	})
	if sealError != nil {
		return "", Unavailable
	}
	defer clear(sealedIdentifier)
	if string(sealedIdentifier) != identifier {
		return "", IdentityMismatch
	}
	return identifier, nil
}

func ClusterIdentity(ctx context.Context) (string, error) {
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return "", connectError
	}
	defer closeConnection(connection)
	return identifier, nil
}

func Ready(ctx context.Context) error {
	connection, _, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	for _, name := range []string{MaintenanceDatabase, "template0", "template1"} {
		if _, revokeError := connection.Exec(ctx, "REVOKE ALL ON DATABASE "+pgIdentifier(name)+" FROM PUBLIC"); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	return nil
}

func Connect(ctx context.Context, record *models.Database, mode SessionMode) (*pgx.Conn, error) {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return nil, roleError
	}
	username := ""
	switch mode {
	case ReadOnly:
		username = roles.Reader
	case ReadWrite:
		username = roles.Writer
	case Objects:
		username = roles.Owner
	default:
		return nil, InvalidConnection
	}
	administrator, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return nil, connectError
	}
	defer closeConnection(administrator)
	if lockError := acquireSessionLock(ctx, administrator, record); lockError != nil {
		return nil, lockError
	}
	if verifyError := verifyDatabase(ctx, administrator, record, roles, identifier, false); verifyError != nil {
		return nil, verifyError
	}
	identity, identityError := verifiedSessionRole(ctx, administrator, record, roles, username)
	if identityError != nil {
		return nil, identityError
	}
	settings, settingsError := databaseSettings(ctx, administrator, record)
	if settingsError != nil {
		return nil, settingsError
	}
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return nil, Unavailable
	}
	password, passwordError := store.Get(SecretNamespace, username)
	if passwordError != nil {
		return nil, Unavailable
	}
	defer clear(password)
	configuration, configurationError := connectionConfiguration(record.Name, username, string(password))
	if configurationError != nil {
		return nil, configurationError
	}
	configuration.RuntimeParams["search_path"] = "public"
	configuration.MaxProtocolMessageBodyLen = 32 << 20
	configuration.BuildContextWatcherHandler = func(connection *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.CancelRequestContextWatcherHandler{Conn: connection, DeadlineDelay: CleanupTimeout}
	}
	limitTransport(configuration)
	configuration.RuntimeParams["statement_timeout"] = strconv.Itoa(settings.StatementTimeoutMilliseconds) + "ms"
	configuration.RuntimeParams["idle_in_transaction_session_timeout"] = strconv.Itoa(settings.IdleTransactionTimeoutMilliseconds) + "ms"
	if mode == ReadOnly {
		configuration.RuntimeParams["default_transaction_read_only"] = "on"
	} else {
		configuration.RuntimeParams["default_transaction_read_only"] = "off"
	}
	connection, targetError := authenticatedConnect(ctx, configuration)
	if targetError != nil {
		return nil, targetError
	}
	if verifyError := validateBackendPID(ctx, administrator, connection, record, roles, identifier, identity); verifyError != nil {
		closeConnection(connection)
		return nil, verifyError
	}
	return connection, nil
}

func verifyDatabase(ctx context.Context, connection *pgx.Conn, record *models.Database, roles Roles, identifier string, allowUnpinned bool) error {
	if record.ClusterID != "" && record.ClusterID != identifier || !allowUnpinned && (record.ClusterID == "" || record.BackendOID == 0) {
		return IdentityMismatch
	}
	var objectOID uint32
	var owner, marker string
	queryError := connection.QueryRow(ctx, `SELECT database.oid, owner.rolname,
		COALESCE(pg_catalog.shobj_description(database.oid, 'pg_database'), '')
		FROM pg_catalog.pg_database AS database JOIN pg_catalog.pg_roles AS owner ON owner.oid = database.datdba
		WHERE database.datname = $1`, record.Name).Scan(&objectOID, &owner, &marker)
	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return InvalidDatabase
		}
		return SafeError(queryError)
	}
	if owner != roles.Custodian || marker != databaseMarker(record) || record.BackendOID != 0 && record.BackendOID != objectOID {
		return IdentityMismatch
	}
	return nil
}

func closeConnection(connection *pgx.Conn) {
	if connection == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), CleanupTimeout)
	defer cancel()
	connection.Close(ctx)
}
