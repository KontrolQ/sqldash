package runtime

import "errors"

var (
	ErrUnsupportedPlatform  = errors.New("managed PostgreSQL is supported only on Linux")
	ErrDisabled             = errors.New("PostgreSQL is disabled")
	ErrClusterLocked        = errors.New("another process owns the PostgreSQL cluster")
	ErrClusterDamaged       = errors.New("PostgreSQL data is incompatible, incomplete or damaged; refusing initialization")
	ErrClusterIdentity      = errors.New("the PostgreSQL cluster identity does not match its protected credentials")
	ErrAdministratorMissing = errors.New("protected PostgreSQL administrator credentials are missing or invalid")
	ErrClusterRunning       = errors.New("PostgreSQL is still running; the cluster lock cannot be released")
	ErrReadinessRequired    = errors.New("PostgreSQL needs an authenticated readiness callback")
)
