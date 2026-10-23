package control

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

type Settings struct {
	ConnectionLimit                    int
	StatementTimeoutMilliseconds       int
	IdleTransactionTimeoutMilliseconds int
}

var InvalidSettings = errors.New("PostgreSQL settings are outside the supported limits")

func GetSettings(ctx context.Context, record *models.Database) (*Settings, error) {
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
	return databaseSettings(ctx, connection, record)
}

func databaseSettings(ctx context.Context, connection *pgx.Conn, record *models.Database) (*Settings, error) {
	settings := &Settings{ConnectionLimit: -1, StatementTimeoutMilliseconds: 30000, IdleTransactionTimeoutMilliseconds: 30000}
	var statementTimeout, idleTimeout string
	queryError := connection.QueryRow(ctx, `SELECT database.datconnlimit,
		COALESCE((SELECT split_part(value, '=', 2) FROM pg_catalog.pg_db_role_setting AS setting,
			unnest(setting.setconfig) AS value WHERE setting.setdatabase = database.oid AND setting.setrole = 0 AND starts_with(value, 'statement_timeout=')), '30000ms'),
		COALESCE((SELECT split_part(value, '=', 2) FROM pg_catalog.pg_db_role_setting AS setting,
			unnest(setting.setconfig) AS value WHERE setting.setdatabase = database.oid AND setting.setrole = 0 AND starts_with(value, 'idle_in_transaction_session_timeout=')), '30000ms')
		FROM pg_catalog.pg_database AS database WHERE database.oid = $1`, record.BackendOID).Scan(&settings.ConnectionLimit, &statementTimeout, &idleTimeout)
	if queryError != nil {
		return nil, SafeError(queryError)
	}
	var parseError error
	settings.StatementTimeoutMilliseconds, parseError = timeoutMilliseconds(statementTimeout)
	if parseError != nil {
		return nil, InvalidSettings
	}
	settings.IdleTransactionTimeoutMilliseconds, parseError = timeoutMilliseconds(idleTimeout)
	if parseError != nil {
		return nil, InvalidSettings
	}
	return settings, nil
}

func timeoutMilliseconds(value string) (int, error) {
	if number, parseError := strconv.Atoi(value); parseError == nil {
		return number, nil
	}
	duration, parseError := time.ParseDuration(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
	if parseError != nil {
		return 0, parseError
	}
	return int(duration / time.Millisecond), nil
}

func applySessionSettings(ctx context.Context, transaction pgx.Tx, record *models.Database, principal string, settings *Settings) error {
	for setting, value := range map[string]int{
		"statement_timeout":                   settings.StatementTimeoutMilliseconds,
		"idle_in_transaction_session_timeout": settings.IdleTransactionTimeoutMilliseconds,
	} {
		if _, alterError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(principal)+" IN DATABASE "+pgIdentifier(record.Name)+" SET "+setting+" = "+pgLiteral(strconv.Itoa(value)+"ms")); alterError != nil {
			return SafeError(alterError)
		}
	}
	return nil
}

func SetSettings(ctx context.Context, record *models.Database, settings Settings) error {
	if settings.ConnectionLimit < -1 || settings.ConnectionLimit == 0 || settings.ConnectionLimit > 10000 ||
		settings.StatementTimeoutMilliseconds < 100 || settings.StatementTimeoutMilliseconds > 300000 ||
		settings.IdleTransactionTimeoutMilliseconds < 100 || settings.IdleTransactionTimeoutMilliseconds > 300000 {
		return InvalidSettings
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
	if lockError := acquireDatabaseLock(ctx, connection, record); lockError != nil {
		return lockError
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	if _, alterError := transaction.Exec(ctx, "ALTER DATABASE "+pgIdentifier(record.Name)+" CONNECTION LIMIT "+strconv.Itoa(settings.ConnectionLimit)); alterError != nil {
		return SafeError(alterError)
	}
	applications, queryError := applicationRoleNames(ctx, transaction, record)
	if queryError != nil {
		return queryError
	}
	principals := append([]string{roles.Owner, roles.Reader, roles.Writer}, applications...)
	for setting, value := range map[string]int{
		"statement_timeout":                   settings.StatementTimeoutMilliseconds,
		"idle_in_transaction_session_timeout": settings.IdleTransactionTimeoutMilliseconds,
	} {
		literal := pgLiteral(strconv.Itoa(value) + "ms")
		if _, alterError := transaction.Exec(ctx, "ALTER DATABASE "+pgIdentifier(record.Name)+" SET "+setting+" = "+literal); alterError != nil {
			return SafeError(alterError)
		}
		for _, principal := range principals {
			if _, alterError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(principal)+" IN DATABASE "+pgIdentifier(record.Name)+" SET "+setting+" = "+literal); alterError != nil {
				return SafeError(alterError)
			}
		}
	}
	return SafeError(transaction.Commit(ctx))
}
