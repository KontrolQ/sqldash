package models

import "time"

type PostgresStatistics struct {
	ID                     uint      `gorm:"primaryKey"`
	DatabaseID             uint      `gorm:"uniqueIndex:postgres_database_sample;index;not null"`
	ClusterID              string    `gorm:"not null"`
	DatabaseOID            uint32    `gorm:"column:database_oid;not null"`
	CollectedAt            time.Time `gorm:"uniqueIndex:postgres_database_sample;index;not null"`
	Version                uint      `gorm:"not null"`
	ResetAt                *time.Time
	PostmasterStartedAt    time.Time
	StatementResetAt       *time.Time
	StatementsDeallocated  int64
	StatementsAvailable    bool
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
	Continuous             bool
	StatementsContinuous   bool
	IntervalStartedAt      *time.Time
	CommittedDelta         int64
	RolledBackDelta        int64
	BlocksReadDelta        int64
	BlocksHitDelta         int64
	DeadlocksDelta         int64
}

type PostgresStatementStatistics struct {
	ID                          uint      `gorm:"primaryKey"`
	DatabaseID                  uint      `gorm:"uniqueIndex:postgres_statement_sample;index;not null"`
	ClusterID                   string    `gorm:"not null"`
	DatabaseOID                 uint32    `gorm:"column:database_oid;not null"`
	NativeOID                   uint32    `gorm:"column:native_oid;uniqueIndex:postgres_statement_sample;not null"`
	QueryID                     string    `gorm:"uniqueIndex:postgres_statement_sample;not null"`
	TopLevel                    bool      `gorm:"uniqueIndex:postgres_statement_sample;not null"`
	CollectedAt                 time.Time `gorm:"uniqueIndex:postgres_statement_sample;index;not null"`
	Version                     uint      `gorm:"not null"`
	StatisticsSince             time.Time
	Statement                   string
	Calls                       int64
	Rows                        int64
	PlanningMs                  float64
	ExecutionMs                 float64
	MinimumExecutionMs          float64
	MaximumExecutionMs          float64
	SharedBlocksHit             int64
	SharedBlocksRead            int64
	LocalBlocksHit              int64
	LocalBlocksRead             int64
	TemporaryBlocksRead         int64
	TemporaryBlocksWritten      int64
	Continuous                  bool
	IntervalStartedAt           *time.Time
	CallsDelta                  int64
	RowsDelta                   int64
	ExecutionDeltaMs            float64
	PlanningDeltaMs             float64
	SharedBlocksHitDelta        int64
	SharedBlocksReadDelta       int64
	LocalBlocksHitDelta         int64
	LocalBlocksReadDelta        int64
	TemporaryBlocksReadDelta    int64
	TemporaryBlocksWrittenDelta int64
}
