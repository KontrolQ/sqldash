package control

import (
	"context"
	"errors"
	"time"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

type StatisticsRecord struct {
	ClusterID              string
	DatabaseOID            uint32
	StorageBytes           int64
	Connections            int64
	TransactionsCommitted  int64
	TransactionsRolledBack int64
	BlocksRead             int64
	BlocksHit              int64
	TuplesReturned         int64
	TuplesFetched          int64
	TuplesInserted         int64
	TuplesUpdated          int64
	TuplesDeleted          int64
	Deadlocks              int64
	TemporaryBytes         int64
	ResetAt                *time.Time
	PostmasterStartedAt    time.Time
	StatementsAvailable    bool
	StatementCalls         int64
	StatementExecutionMs   float64
	StatementRows          int64
	StatementResetAt       *time.Time
	StatementsDeallocated  int64
}

type StatementRecord struct {
	NativeOID              uint32
	QueryID                string
	Query                  string
	Calls                  int64
	Rows                   int64
	PlanningMs             float64
	ExecutionMs            float64
	MinimumExecutionMs     float64
	MaximumExecutionMs     float64
	MeanExecutionMs        float64
	SharedBlocksHit        int64
	SharedBlocksRead       int64
	LocalBlocksHit         int64
	LocalBlocksRead        int64
	TemporaryBlocksRead    int64
	TemporaryBlocksWritten int64
	TopLevel               bool
	StatisticsSince        time.Time
}

func Statistics(ctx context.Context, record *models.Database) (*StatisticsRecord, error) {
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
	statistics := &StatisticsRecord{ClusterID: identifier, DatabaseOID: record.BackendOID}
	queryError := connection.QueryRow(ctx, `SELECT pg_catalog.pg_database_size(datid), numbackends::bigint,
		xact_commit, xact_rollback, blks_read, blks_hit, tup_returned, tup_fetched, tup_inserted, tup_updated,
		tup_deleted, deadlocks, temp_bytes, stats_reset, pg_catalog.pg_postmaster_start_time()
		FROM pg_catalog.pg_stat_database WHERE datid = $1`, record.BackendOID).Scan(&statistics.StorageBytes,
		&statistics.Connections, &statistics.TransactionsCommitted, &statistics.TransactionsRolledBack,
		&statistics.BlocksRead, &statistics.BlocksHit, &statistics.TuplesReturned, &statistics.TuplesFetched,
		&statistics.TuplesInserted, &statistics.TuplesUpdated, &statistics.TuplesDeleted, &statistics.Deadlocks,
		&statistics.TemporaryBytes, &statistics.ResetAt, &statistics.PostmasterStartedAt)
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	schema, extensionError := statementStatisticsSchema(ctx, connection)
	if extensionError != nil {
		return nil, extensionError
	}
	if schema == "" {
		return statistics, nil
	}
	queryError = connection.QueryRow(ctx, "SELECT COALESCE(sum(statement.calls), 0)::bigint, COALESCE(sum(statement.total_exec_time), 0)::double precision, COALESCE(sum(statement.rows), 0)::bigint FROM "+pgx.Identifier{schema, "pg_stat_statements"}.Sanitize()+" AS statement JOIN pg_catalog.pg_roles AS role ON role.oid = statement.userid WHERE statement.dbid = $1 AND statement.toplevel AND pg_catalog.shobj_description(role.oid, 'pg_authid') = $2 || role.rolname AND (role.rolname = ANY($3::text[]) OR starts_with(role.rolname, $4)) AND statement.query !~* $5", record.BackendOID, databaseMarker(record)+":", telemetryRoleNames(roles, record.Name), applicationRolePrefix(roles), SensitiveStatisticsPattern).Scan(&statistics.StatementCalls, &statistics.StatementExecutionMs, &statistics.StatementRows)
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	queryError = connection.QueryRow(ctx, "SELECT stats_reset, dealloc FROM "+pgx.Identifier{schema, "pg_stat_statements_info"}.Sanitize()).Scan(&statistics.StatementResetAt, &statistics.StatementsDeallocated)
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	statistics.StatementsAvailable = true
	return statistics, nil
}

func Statements(ctx context.Context, record *models.Database) ([]StatementRecord, error) {
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
	schema, extensionError := statementStatisticsSchema(ctx, connection)
	if extensionError != nil {
		return nil, extensionError
	}
	if schema == "" {
		return nil, nil
	}
	query := "SELECT statement.userid, statement.queryid::text, statement.query, statement.calls, statement.rows, statement.total_plan_time, statement.total_exec_time, statement.min_exec_time, statement.max_exec_time, statement.mean_exec_time, statement.shared_blks_hit, statement.shared_blks_read, statement.local_blks_hit, statement.local_blks_read, statement.temp_blks_read, statement.temp_blks_written, statement.toplevel, statement.stats_since FROM " + pgx.Identifier{schema, "pg_stat_statements"}.Sanitize() + " AS statement JOIN pg_catalog.pg_roles AS role ON role.oid = statement.userid WHERE statement.dbid = $1 AND pg_catalog.shobj_description(role.oid, 'pg_authid') = $2 || role.rolname AND (role.rolname = ANY($3::text[]) OR starts_with(role.rolname, $4)) AND statement.query !~* $5 ORDER BY statement.total_exec_time DESC"
	rows, queryError := connection.Query(ctx, query, record.BackendOID, databaseMarker(record)+":", telemetryRoleNames(roles, record.Name), applicationRolePrefix(roles), SensitiveStatisticsPattern)
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	defer rows.Close()
	statements := []StatementRecord{}
	for rows.Next() {
		var statement StatementRecord
		if scanError := rows.Scan(&statement.NativeOID, &statement.QueryID, &statement.Query, &statement.Calls, &statement.Rows,
			&statement.PlanningMs, &statement.ExecutionMs, &statement.MinimumExecutionMs, &statement.MaximumExecutionMs,
			&statement.MeanExecutionMs, &statement.SharedBlocksHit, &statement.SharedBlocksRead, &statement.LocalBlocksHit,
			&statement.LocalBlocksRead, &statement.TemporaryBlocksRead, &statement.TemporaryBlocksWritten, &statement.TopLevel, &statement.StatisticsSince); scanError != nil {
			return nil, SafeError(scanError)
		}
		statements = append(statements, statement)
	}
	return statements, SafeError(rows.Err())
}

func statementStatisticsSchema(ctx context.Context, connection *pgx.Conn) (string, error) {
	var schema string
	queryError := connection.QueryRow(ctx, `SELECT namespace.nspname FROM pg_catalog.pg_extension AS extension
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = extension.extnamespace WHERE extension.extname = 'pg_stat_statements'`).Scan(&schema)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return "", nil
	}
	return schema, SafeError(queryError)
}
