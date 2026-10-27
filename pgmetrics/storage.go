package pgmetrics

import (
	"context"
	"errors"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/pgcluster/control"

	"gorm.io/gorm"
)

func persist(ctx context.Context, record *models.Database, statistics *control.StatisticsRecord, statements []control.StatementRecord, collectedAt time.Time) error {
	return database.DB.WithContext(ctx).Transaction(func(store *gorm.DB) error {
		var registration models.Database
		identity := models.Database{Model: gorm.Model{ID: record.ID}, Engine: models.EnginePostgres, BackendOID: record.BackendOID, ClusterID: record.ClusterID}
		if err := store.Where(&identity).Where("state IN ?", []string{models.DatabasePending, models.DatabaseApplying, models.DatabaseReady}).First(&registration).Error; err != nil {
			return IdentityMismatch
		}
		previous := models.PostgresStatistics{}
		err := store.Where("database_id = ?", record.ID).Order("collected_at DESC").First(&previous).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		current := databaseSample(record, statistics, collectedAt)
		current.Continuous = previous.ID != 0 && previous.CollectedAt.Before(collectedAt) && previous.Version == SampleVersion && previous.ClusterID == current.ClusterID &&
			previous.DatabaseOID == current.DatabaseOID && sameTime(previous.ResetAt, current.ResetAt) &&
			previous.PostmasterStartedAt.Equal(current.PostmasterStartedAt) && monotonicDatabase(current, previous)
		current.StatementsContinuous = current.Continuous && previous.StatementsAvailable && current.StatementsAvailable &&
			sameTime(previous.StatementResetAt, current.StatementResetAt) && previous.StatementsDeallocated == current.StatementsDeallocated
		if current.Continuous {
			current.IntervalStartedAt = &previous.CollectedAt
			current.CommittedDelta = current.TransactionsCommitted - previous.TransactionsCommitted
			current.RolledBackDelta = current.TransactionsRolledBack - previous.TransactionsRolledBack
			current.BlocksReadDelta = current.BlocksRead - previous.BlocksRead
			current.BlocksHitDelta = current.BlocksHit - previous.BlocksHit
			current.DeadlocksDelta = current.Deadlocks - previous.Deadlocks
		}
		if err := store.Create(&current).Error; err != nil {
			return err
		}
		for _, statement := range statements {
			sample := statementSample(record, statement, collectedAt)
			var before models.PostgresStatementStatistics
			err := store.Where("database_id = ? AND native_oid = ? AND query_id = ? AND top_level = ?", record.ID, sample.NativeOID, sample.QueryID, sample.TopLevel).Order("collected_at DESC").First(&before).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			sample.Continuous = current.StatementsContinuous && before.ID != 0 && before.Version == SampleVersion &&
				before.ClusterID == sample.ClusterID && before.DatabaseOID == sample.DatabaseOID &&
				before.CollectedAt.Equal(previous.CollectedAt) && before.StatisticsSince.Equal(sample.StatisticsSince) &&
				before.Statement == sample.Statement && monotonicStatement(sample, before)
			if current.StatementsContinuous && !sample.StatisticsSince.Before(previous.CollectedAt) && !sample.StatisticsSince.After(collectedAt) && (before.ID == 0 || !before.StatisticsSince.Equal(sample.StatisticsSince)) {
				before = models.PostgresStatementStatistics{}
				sample.Continuous = true
			}
			if sample.Continuous {
				sample.IntervalStartedAt = &previous.CollectedAt
				sample.CallsDelta = sample.Calls - before.Calls
				sample.RowsDelta = sample.Rows - before.Rows
				sample.ExecutionDeltaMs = sample.ExecutionMs - before.ExecutionMs
				sample.PlanningDeltaMs = sample.PlanningMs - before.PlanningMs
				sample.SharedBlocksHitDelta = sample.SharedBlocksHit - before.SharedBlocksHit
				sample.SharedBlocksReadDelta = sample.SharedBlocksRead - before.SharedBlocksRead
				sample.LocalBlocksHitDelta = sample.LocalBlocksHit - before.LocalBlocksHit
				sample.LocalBlocksReadDelta = sample.LocalBlocksRead - before.LocalBlocksRead
				sample.TemporaryBlocksReadDelta = sample.TemporaryBlocksRead - before.TemporaryBlocksRead
				sample.TemporaryBlocksWrittenDelta = sample.TemporaryBlocksWritten - before.TemporaryBlocksWritten
			}
			if err := store.Create(&sample).Error; err != nil {
				return err
			}
		}
		if config.History.Retention == 0 {
			return nil
		}
		cutoff := collectedAt.Add(-config.History.Retention)
		if err := store.Where("collected_at < ?", cutoff).Delete(&models.PostgresStatementStatistics{}).Error; err != nil {
			return err
		}
		return store.Where("collected_at < ?", cutoff).Delete(&models.PostgresStatistics{}).Error
	})
}

func DeleteForDatabase(databaseID uint) error {
	if databaseID == 0 {
		return IdentityMismatch
	}
	sampleLock.Lock()
	defer sampleLock.Unlock()
	return database.DB.Transaction(func(store *gorm.DB) error {
		if err := store.Where("database_id = ?", databaseID).Delete(&models.PostgresStatementStatistics{}).Error; err != nil {
			return err
		}
		return store.Where("database_id = ?", databaseID).Delete(&models.PostgresStatistics{}).Error
	})
}

func databaseSample(record *models.Database, statistics *control.StatisticsRecord, collectedAt time.Time) models.PostgresStatistics {
	return models.PostgresStatistics{
		DatabaseID: record.ID, ClusterID: record.ClusterID, DatabaseOID: record.BackendOID, CollectedAt: collectedAt, Version: SampleVersion,
		ResetAt: statistics.ResetAt, PostmasterStartedAt: statistics.PostmasterStartedAt, StatementResetAt: statistics.StatementResetAt,
		StatementsDeallocated: statistics.StatementsDeallocated, StatementsAvailable: statistics.StatementsAvailable,
		StorageBytes: statistics.StorageBytes, Connections: statistics.Connections, TransactionsCommitted: statistics.TransactionsCommitted,
		TransactionsRolledBack: statistics.TransactionsRolledBack, BlocksRead: statistics.BlocksRead, BlocksHit: statistics.BlocksHit,
		TuplesReturned: statistics.TuplesReturned, TuplesFetched: statistics.TuplesFetched, TuplesInserted: statistics.TuplesInserted,
		TuplesUpdated: statistics.TuplesUpdated, TuplesDeleted: statistics.TuplesDeleted, Deadlocks: statistics.Deadlocks, TemporaryBytes: statistics.TemporaryBytes,
	}
}

func statementSample(record *models.Database, statement control.StatementRecord, collectedAt time.Time) models.PostgresStatementStatistics {
	return models.PostgresStatementStatistics{
		DatabaseID: record.ID, ClusterID: record.ClusterID, DatabaseOID: record.BackendOID, NativeOID: statement.NativeOID,
		QueryID: statement.QueryID, TopLevel: statement.TopLevel, CollectedAt: collectedAt, Version: SampleVersion, StatisticsSince: statement.StatisticsSince,
		Statement: statement.Query, Calls: statement.Calls, Rows: statement.Rows, PlanningMs: statement.PlanningMs, ExecutionMs: statement.ExecutionMs,
		MinimumExecutionMs: statement.MinimumExecutionMs, MaximumExecutionMs: statement.MaximumExecutionMs,
		SharedBlocksHit: statement.SharedBlocksHit, SharedBlocksRead: statement.SharedBlocksRead, LocalBlocksHit: statement.LocalBlocksHit,
		LocalBlocksRead: statement.LocalBlocksRead, TemporaryBlocksRead: statement.TemporaryBlocksRead, TemporaryBlocksWritten: statement.TemporaryBlocksWritten,
	}
}

func monotonicDatabase(now models.PostgresStatistics, before models.PostgresStatistics) bool {
	return now.TransactionsCommitted >= before.TransactionsCommitted && now.TransactionsRolledBack >= before.TransactionsRolledBack &&
		now.BlocksRead >= before.BlocksRead && now.BlocksHit >= before.BlocksHit && now.Deadlocks >= before.Deadlocks &&
		now.TuplesReturned >= before.TuplesReturned && now.TuplesFetched >= before.TuplesFetched && now.TuplesInserted >= before.TuplesInserted &&
		now.TuplesUpdated >= before.TuplesUpdated && now.TuplesDeleted >= before.TuplesDeleted && now.TemporaryBytes >= before.TemporaryBytes
}

func monotonicStatement(now models.PostgresStatementStatistics, before models.PostgresStatementStatistics) bool {
	return now.Calls >= before.Calls && now.Rows >= before.Rows && now.ExecutionMs >= before.ExecutionMs && now.PlanningMs >= before.PlanningMs &&
		now.SharedBlocksHit >= before.SharedBlocksHit && now.SharedBlocksRead >= before.SharedBlocksRead && now.LocalBlocksHit >= before.LocalBlocksHit &&
		now.LocalBlocksRead >= before.LocalBlocksRead && now.TemporaryBlocksRead >= before.TemporaryBlocksRead && now.TemporaryBlocksWritten >= before.TemporaryBlocksWritten
}
