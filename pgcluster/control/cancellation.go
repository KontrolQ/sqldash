package control

import (
	"context"
	"errors"
	"time"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

func CancelActivity(ctx context.Context, record *models.Database, target ActivityTarget) error {
	return manageActivity(ctx, record, target, false)
}

func TerminateActivity(ctx context.Context, record *models.Database, target ActivityTarget) error {
	return manageActivity(ctx, record, target, true)
}

func manageActivity(ctx context.Context, record *models.Database, target ActivityTarget, terminate bool) error {
	if target.PID == 0 || target.StartedAt.IsZero() {
		return InvalidConnection
	}
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
	var actualDatabase uint32
	var username string
	var startedAt time.Time
	queryError := connection.QueryRow(ctx, "SELECT datid, usename, backend_start FROM pg_catalog.pg_stat_activity WHERE pid = $1", int64(target.PID)).Scan(&actualDatabase, &username, &startedAt)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return nil
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	if actualDatabase != record.BackendOID || !startedAt.Equal(target.StartedAt) {
		return IdentityMismatch
	}
	if roleError := managedSessionRole(ctx, connection, record, username); roleError != nil {
		return roleError
	}
	if terminate {
		return terminateMatchedSession(ctx, connection, record, username, target.PID, target.StartedAt)
	}
	var canceled bool
	queryError = connection.QueryRow(ctx, `SELECT pg_catalog.pg_cancel_backend(pid) FROM pg_catalog.pg_stat_activity
		WHERE pid = $1 AND datid = $2 AND usename = $3 AND backend_start = $4`, int64(target.PID), record.BackendOID, username, target.StartedAt).Scan(&canceled)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return nil
	}
	if queryError != nil {
		return SafeError(queryError)
	}
	if !canceled {
		return TerminationUnconfirmed
	}
	return nil
}
