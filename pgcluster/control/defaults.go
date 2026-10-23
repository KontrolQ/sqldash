package control

import "time"

const (
	LogPrefix              = "postgres-control"
	MaintenanceDatabase    = "postgres"
	SecretNamespace        = "postgres-roles"
	ClusterNamespace       = "postgres-control"
	ClusterIdentitySecret  = "cluster-identity"
	RolePrefix             = "sqd_"
	StatementTimeout       = 30 * time.Second
	ConnectionTimeout      = 5 * time.Second
	CleanupTimeout         = 5 * time.Second
	MaximumIdentifierBytes = 63
	PasswordBytes          = 32
	SCRAMIterations        = 4096
)

type SessionMode string

const (
	ReadOnly  SessionMode = "read-only"
	ReadWrite SessionMode = "read-write"
	Objects   SessionMode = "objects"
)
