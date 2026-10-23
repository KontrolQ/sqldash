package postgres

import (
	"context"
	"errors"
	"fmt"

	"sqldash/pgcluster/control"
	"sqldash/sqld"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
)

var (
	ResultLimit     = errors.New("the PostgreSQL result exceeds the browser response limit")
	InvalidRelation = errors.New("the PostgreSQL relation was not found")
	InvalidValue    = errors.New("the PostgreSQL value could not be decoded")
)

func statementError(index int, operationError error) error {
	var oversizedMessage *pgproto3.ExceededMaxBodyLenErr
	if errors.Is(operationError, control.ResponseLimit) || errors.As(operationError, &oversizedMessage) {
		return ResultLimit
	}
	if errors.Is(operationError, context.Canceled) || errors.Is(operationError, context.DeadlineExceeded) {
		return operationError
	}
	var postgresError *pgconn.PgError
	if errors.As(operationError, &postgresError) {
		return &sqld.StatementError{At: index, Message: fmt.Sprintf("%s (%s)", postgresError.Message, postgresError.Code)}
	}
	return control.SafeError(operationError)
}
