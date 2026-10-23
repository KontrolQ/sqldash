package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	Unavailable                  = errors.New("PostgreSQL is unavailable")
	InvalidDatabase              = errors.New("the PostgreSQL database identity is invalid")
	IdentityMismatch             = errors.New("the PostgreSQL cluster or database identity does not match")
	InvalidConnection            = errors.New("the PostgreSQL connection identity or scope is invalid")
	InvalidPassword              = errors.New("a PostgreSQL password is required")
	RoleMismatch                 = errors.New("the PostgreSQL role does not belong to this database")
	VersionMismatch              = errors.New("PostgreSQL 18 is required")
	TerminationUnconfirmed       = errors.New("the PostgreSQL session termination could not be confirmed")
	PolicyUnconfirmed            = errors.New("PostgreSQL privileges could not be reconciled")
	AuthenticationPolicyMismatch = errors.New("PostgreSQL SCRAM password authentication is required")
)

type ExtensionDependencyError struct {
	Name string
}

func (self *ExtensionDependencyError) Error() string {
	return fmt.Sprintf("enable the %s extension in management first", self.Name)
}

type Error struct {
	Code    string
	Message string
}

func (self *Error) Error() string {
	return self.Message
}

func SafeError(operationError error) error {
	if operationError == nil {
		return nil
	}
	if errors.Is(operationError, context.Canceled) || errors.Is(operationError, context.DeadlineExceeded) {
		return operationError
	}
	var postgresError *pgconn.PgError
	if errors.As(operationError, &postgresError) {
		return &Error{Code: postgresError.Code, Message: fmt.Sprintf("PostgreSQL operation failed (%s)", postgresError.Code)}
	}
	return Unavailable
}
