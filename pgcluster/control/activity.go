package control

import (
	"context"
	"errors"
	"strings"
	"time"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

type ActivityRecord struct {
	PID                  uint32
	DatabaseOID          uint32
	NativeOID            uint32
	Username             string
	Application          string
	ClientAddress        *string
	StartedAt            time.Time
	TransactionStartedAt *time.Time
	QueryStartedAt       *time.Time
	StateChangedAt       *time.Time
	State                string
	WaitType             string
	WaitEvent            string
	Query                string
	Manageable           bool
}

type ActivityTarget struct {
	PID       uint32
	StartedAt time.Time
}

func Activity(ctx context.Context, record *models.Database) ([]ActivityRecord, error) {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return nil, roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return nil, connectError
	}
	defer closeConnection(connection)
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return nil, verifyError
	}
	rows, queryError := connection.Query(ctx, `SELECT activity.pid::bigint, activity.datid, activity.usesysid, activity.usename, activity.application_name, activity.client_addr::text,
		activity.backend_start, activity.xact_start, activity.query_start, activity.state_change, COALESCE(activity.state, ''), COALESCE(activity.wait_event_type, ''),
		COALESCE(activity.wait_event, ''), COALESCE(activity.query, '') FROM pg_catalog.pg_stat_activity AS activity
		JOIN pg_catalog.pg_roles AS role ON role.oid = activity.usesysid
		WHERE activity.datid = $1 AND activity.backend_type = 'client backend'
		AND pg_catalog.shobj_description(role.oid, 'pg_authid') = $2 || role.rolname
		AND (role.rolname = ANY($3::text[]) OR starts_with(role.rolname, $4))
		ORDER BY activity.backend_start, activity.pid`, record.BackendOID, databaseMarker(record)+":", telemetryRoleNames(roles, record.Name), applicationRolePrefix(roles))
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	defer rows.Close()
	activity := []ActivityRecord{}
	for rows.Next() {
		var entry ActivityRecord
		if scanError := rows.Scan(&entry.PID, &entry.DatabaseOID, &entry.NativeOID, &entry.Username, &entry.Application,
			&entry.ClientAddress, &entry.StartedAt, &entry.TransactionStartedAt, &entry.QueryStartedAt, &entry.StateChangedAt,
			&entry.State, &entry.WaitType, &entry.WaitEvent, &entry.Query); scanError != nil {
			return nil, SafeError(scanError)
		}
		entry.Query = activityQuery(entry.Query)
		entry.Manageable = true
		activity = append(activity, entry)
	}
	return activity, SafeError(rows.Err())
}

func managedSessionRole(ctx context.Context, connection *pgx.Conn, record *models.Database, username string) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	prefix := strings.TrimSuffix(roles.Custodian, "custodian")
	if username != record.Name && username != roles.Owner && username != roles.Reader && username != roles.Writer && !strings.HasPrefix(username, prefix+"app_") && !strings.HasPrefix(username, prefix+"transfer_") {
		return RoleMismatch
	}
	var marker string
	queryError := connection.QueryRow(ctx, `SELECT COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), '') FROM pg_catalog.pg_roles WHERE rolname = $1`, username).Scan(&marker)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return nil
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	if marker != roleMarker(record, username) {
		return RoleMismatch
	}
	return nil
}

func Terminate(ctx context.Context, record *models.Database, username string, pid uint32) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	if record.BackendOID == 0 || record.ClusterID == "" || record.ClusterID != identifier {
		return IdentityMismatch
	}
	if roleError := managedSessionRole(ctx, connection, record, username); roleError != nil {
		return roleError
	}
	var present bool
	if inspectError := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_stat_activity WHERE pid = $1)", int64(pid)).Scan(&present); inspectError != nil {
		return SafeError(inspectError)
	}
	if !present {
		return nil
	}
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		if errors.Is(verifyError, InvalidDatabase) {
			if inspectError := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_stat_activity WHERE pid = $1)", int64(pid)).Scan(&present); inspectError != nil {
				return SafeError(inspectError)
			}
			if !present {
				return nil
			}
		}
		return verifyError
	}
	return terminateSession(ctx, connection, record, username, pid)
}

func terminateSession(ctx context.Context, connection *pgx.Conn, record *models.Database, username string, pid uint32) error {
	var databaseOID uint32
	var actualUsername string
	var startedAt time.Time
	queryError := connection.QueryRow(ctx, `SELECT datid, usename, backend_start FROM pg_catalog.pg_stat_activity WHERE pid = $1`, int64(pid)).Scan(&databaseOID, &actualUsername, &startedAt)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return nil
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	if databaseOID != record.BackendOID || actualUsername != username {
		return RoleMismatch
	}
	return terminateMatchedSession(ctx, connection, record, username, pid, startedAt)
}

func terminateMatchedSession(ctx context.Context, connection *pgx.Conn, record *models.Database, username string, pid uint32, startedAt time.Time) error {
	var terminated bool
	queryError := connection.QueryRow(ctx, `SELECT pg_catalog.pg_terminate_backend(pid, 5000) FROM pg_catalog.pg_stat_activity
		WHERE pid = $1 AND datid = $2 AND usename = $3 AND backend_start = $4`, int64(pid), record.BackendOID, username, startedAt).Scan(&terminated)
	if queryError != nil && !errors.Is(queryError, pgx.ErrNoRows) {
		return SafeError(queryError)
	}
	var remaining bool
	if confirmError := connection.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_stat_activity
		WHERE pid = $1 AND backend_start = $2)`, int64(pid), startedAt).Scan(&remaining); confirmError != nil {
		return SafeError(confirmError)
	}
	if remaining {
		return TerminationUnconfirmed
	}
	return nil
}

func terminateLogin(ctx context.Context, record *models.Database, username string) error {
	roles, roleError := DatabaseRoles(record)
	if roleError != nil {
		return roleError
	}
	connection, identifier, connectError := privateAdminConnect(ctx, MaintenanceDatabase)
	if connectError != nil {
		return connectError
	}
	defer closeConnection(connection)
	if verifyError := verifyDatabase(ctx, connection, record, roles, identifier, false); verifyError != nil {
		return verifyError
	}
	return terminateRoleSessions(ctx, connection, record, username)
}

func terminateRoleSessions(ctx context.Context, connection *pgx.Conn, record *models.Database, username string) error {
	if roleError := managedSessionRole(ctx, connection, record, username); roleError != nil {
		return roleError
	}
	rows, queryError := connection.Query(ctx, "SELECT pid::bigint FROM pg_catalog.pg_stat_activity WHERE usename = $1", username)
	if queryError != nil {
		return SafeError(queryError)
	}
	pids := []uint32{}
	for rows.Next() {
		var pid uint32
		if scanError := rows.Scan(&pid); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		pids = append(pids, pid)
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, pid := range pids {
		if terminateError := terminateSession(ctx, connection, record, username, pid); terminateError != nil {
			return terminateError
		}
	}
	var remaining bool
	if confirmError := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_stat_activity WHERE usename = $1)", username).Scan(&remaining); confirmError != nil {
		return SafeError(confirmError)
	}
	if remaining {
		return TerminationUnconfirmed
	}
	return nil
}
