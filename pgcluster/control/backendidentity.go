package control

import (
	"context"
	"errors"
	"net"
	"time"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

type sessionRoleIdentity struct {
	Name string
	OID  uint32
}

func acquireSessionLock(ctx context.Context, administrator *pgx.Conn, record *models.Database) error {
	_, lockError := administrator.Exec(ctx, "SELECT pg_catalog.pg_advisory_lock_shared(hashtextextended($1, 0))", databaseMarker(record))
	return SafeError(lockError)
}

func verifiedSessionRole(ctx context.Context, administrator *pgx.Conn, record *models.Database, roles Roles, username string) (*sessionRoleIdentity, error) {
	if username != roles.Observer {
		if roleError := managedSessionRole(ctx, administrator, record, username); roleError != nil {
			return nil, roleError
		}
	}
	identity := &sessionRoleIdentity{Name: username}
	var marker string
	var login, superuser, createDatabase, createRole, replication, bypassRLS bool
	queryError := administrator.QueryRow(ctx, `SELECT oid, COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), ''),
		rolcanlogin, rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls
		FROM pg_catalog.pg_roles WHERE rolname = $1`, username).Scan(&identity.OID, &marker,
		&login, &superuser, &createDatabase, &createRole, &replication, &bypassRLS)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return nil, RoleMismatch
	}
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	if identity.OID == 0 || marker != roleMarker(record, username) || !login || superuser || createDatabase || createRole || replication || bypassRLS {
		return nil, RoleMismatch
	}
	return identity, nil
}

func validateBackendPID(ctx context.Context, administrator, connection *pgx.Conn, record *models.Database, roles Roles, identifier string, expected *sessionRoleIdentity) error {
	if connection == nil || connection.IsClosed() || expected == nil || connection.PgConn().PID() == 0 {
		return IdentityMismatch
	}
	clientAddress, clientOK := connection.PgConn().Conn().LocalAddr().(*net.TCPAddr)
	serverAddress, serverOK := connection.PgConn().Conn().RemoteAddr().(*net.TCPAddr)
	adminAddress, adminOK := administrator.PgConn().Conn().RemoteAddr().(*net.TCPAddr)
	if !clientOK || !serverOK || !adminOK || !serverAddress.IP.IsLoopback() ||
		!serverAddress.IP.Equal(adminAddress.IP) || serverAddress.Port != adminAddress.Port {
		return IdentityMismatch
	}
	var databaseOID, roleOID uint32
	var actualRole, address, backendType string
	var port int
	var startedAt time.Time
	queryError := administrator.QueryRow(ctx, `SELECT datid, usesysid, usename, pg_catalog.host(client_addr),
		client_port, backend_start, backend_type FROM pg_catalog.pg_stat_activity WHERE pid = $1`,
		int64(connection.PgConn().PID())).Scan(&databaseOID, &roleOID, &actualRole, &address, &port, &startedAt, &backendType)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return IdentityMismatch
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	// The live socket disambiguates reused PIDs without trusting clocks on either endpoint.
	if databaseOID != record.BackendOID || roleOID != expected.OID || actualRole != expected.Name ||
		backendType != "client backend" || startedAt.IsZero() || port != clientAddress.Port || !clientAddress.IP.Equal(net.ParseIP(address)) {
		return IdentityMismatch
	}
	actual, roleError := verifiedSessionRole(ctx, administrator, record, roles, expected.Name)
	if roleError != nil {
		return roleError
	}
	if actual.OID != expected.OID {
		return IdentityMismatch
	}
	return verifyDatabase(ctx, administrator, record, roles, identifier, false)
}
