package postgres

import (
	"context"
	"errors"
	"time"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

func (self *Backend) Run(ctx context.Context, record *models.Database, mode engines.Mode, statements ...engines.Statement) ([]*engines.Result, error) {
	queryContext, cancel := context.WithCancel(ctx)
	defer cancel()
	connection, connectError := Connect(queryContext, record, mode)
	if connectError != nil {
		return nil, connectError
	}
	defer closeSession(connection)
	maximumBytes := config.Query.MaximumResultBytes
	if maximumBytes == 0 {
		maximumBytes = -1
	}
	if limitError := control.LimitResponseBytes(connection, maximumBytes); limitError != nil {
		return nil, limitError
	}
	guarded := false
	for _, statement := range statements {
		if len(statement.Arguments) > 0 || statement.ExpectedAffected != nil {
			guarded = true
			break
		}
	}
	var results []*engines.Result
	var runError error
	if guarded {
		results, runError = runAtomic(queryContext, connection, statements)
	} else {
		results, runError = runConsole(queryContext, connection, statements)
	}
	control.LimitResponseBytes(connection, -1)
	if connection.PgConn().TxStatus() != 'I' {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), control.CleanupTimeout)
		_, rollbackError := connection.Exec(cleanupContext, "ROLLBACK")
		cleanupCancel()
		if runError == nil && rollbackError != nil {
			runError = control.SafeError(rollbackError)
		}
	}
	closeSession(connection)
	if mode == engines.Objects {
		policyContext, policyCancel := context.WithTimeout(context.WithoutCancel(ctx), control.StatementTimeout)
		policyError := control.ReconcilePolicy(policyContext, record)
		policyCancel()
		if policyError != nil {
			return nil, errors.Join(engines.PolicyUnconfirmed, control.PolicyUnconfirmed, policyError)
		}
	}
	return results, runError
}

func runConsole(ctx context.Context, connection *pgx.Conn, statements []engines.Statement) ([]*engines.Result, error) {
	results := []*engines.Result{}
	for index, statement := range statements {
		startedAt := time.Now()
		rawResults, queryError := connection.PgConn().Exec(ctx, statement.SQL).ReadAll()
		if queryError != nil {
			return nil, statementError(index, queryError)
		}
		for _, rawResult := range rawResults {
			if rawResult.Err != nil {
				return nil, statementError(index, rawResult.Err)
			}
			result, normalizeError := normalizeResult(connection.TypeMap(), rawResult.FieldDescriptions, rawResult.Rows)
			if normalizeError != nil {
				return nil, normalizeError
			}
			result.Affected = modifiedRows(rawResult.CommandTag)
			if len(rawResults) == 1 {
				result.DurationMs = float64(time.Since(startedAt)) / float64(time.Millisecond)
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func runAtomic(ctx context.Context, connection *pgx.Conn, statements []engines.Statement) ([]*engines.Result, error) {
	if validateError := validateAtomicStatements(statements); validateError != nil {
		return nil, validateError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return nil, control.SafeError(beginError)
	}
	defer rollbackSession(transaction)
	results := []*engines.Result{}
	for index, statement := range statements {
		startedAt := time.Now()
		arguments := append([]any{pgx.QueryResultFormats{pgx.TextFormatCode}}, statement.Arguments...)
		rows, queryError := transaction.Query(ctx, statement.SQL, arguments...)
		if queryError != nil {
			return nil, statementError(index, queryError)
		}
		result := &engines.Result{}
		fields := rows.FieldDescriptions()
		result.Columns = resultColumns(connection.TypeMap(), fields)
		totalBytes := 0
		for rows.Next() {
			values := rows.RawValues()
			row, rowBytes, normalizeError := normalizeRow(connection.TypeMap(), fields, values)
			if normalizeError != nil {
				rows.Close()
				return nil, normalizeError
			}
			totalBytes += rowBytes
			if config.Query.MaximumRows > 0 && len(result.Rows) >= config.Query.MaximumRows || config.Query.MaximumResultBytes > 0 && int64(totalBytes) > config.Query.MaximumResultBytes {
				rows.Close()
				return nil, ResultLimit
			}
			result.Rows = append(result.Rows, row)
		}
		queryError = rows.Err()
		rows.Close()
		if queryError != nil {
			return nil, statementError(index, queryError)
		}
		result.Affected = modifiedRows(rows.CommandTag())
		if statement.ExpectedAffected != nil && rows.CommandTag().RowsAffected() != *statement.ExpectedAffected {
			return nil, engines.Conflict
		}
		if connection.PgConn().TxStatus() != 'T' {
			return nil, engines.Conflict
		}
		result.DurationMs = float64(time.Since(startedAt)) / float64(time.Millisecond)
		results = append(results, result)
	}
	if limitError := control.LimitResponseBytes(connection, -1); limitError != nil {
		return nil, limitError
	}
	if commitError := transaction.Commit(ctx); commitError != nil {
		return nil, control.SafeError(commitError)
	}
	return results, nil
}

func closeSession(connection *pgx.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), control.CleanupTimeout)
	defer cancel()
	connection.Close(ctx)
}

func rollbackSession(transaction pgx.Tx) {
	control.LimitResponseBytes(transaction.Conn(), -1)
	ctx, cancel := context.WithTimeout(context.Background(), control.CleanupTimeout)
	defer cancel()
	transaction.Rollback(ctx)
}
