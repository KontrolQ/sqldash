package pgmetrics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

type QuerySummary struct {
	DatabaseID             uint
	DatabaseName           string
	ClusterID              string
	DatabaseOID            uint32 `gorm:"column:database_oid"`
	NativeOID              uint32 `gorm:"column:native_oid"`
	QueryID                string
	TopLevel               bool
	StatisticsSince        time.Time
	Statement              string
	Calls                  int64
	Rows                   int64
	ExecutionMs            float64
	PlanningMs             float64
	MinimumExecutionMs     float64
	MaximumExecutionMs     float64
	SharedBlocksHit        int64
	SharedBlocksRead       int64
	LocalBlocksHit         int64
	LocalBlocksRead        int64
	TemporaryBlocksRead    int64
	TemporaryBlocksWritten int64
}

func (self QuerySummary) Key() string {
	return fmt.Sprintf("postgres:%d:%s:%d:%d:%s:%t:%s", self.DatabaseID, self.ClusterID, self.DatabaseOID, self.NativeOID, self.QueryID, self.TopLevel, self.StatisticsSince.UTC().Format(time.RFC3339Nano))
}

func (self QuerySummary) MeanExecutionMs() float64 {
	if self.Calls == 0 {
		return 0
	}
	return self.ExecutionMs / float64(self.Calls)
}

type History struct {
	Latest           []models.PostgresStatistics
	Queries          []QuerySummary
	Calls            int64
	ExecutionMs      float64
	Rows             int64
	SharedBlocksHit  int64
	SharedBlocksRead int64
	Committed        int64
	RolledBack       int64
	Deadlocks        int64
	Discontinuities  int64
	Intervals        int64
	FirstCollectedAt *time.Time
	LastCollectedAt  *time.Time
}

func ReadHistory(ctx context.Context, records []models.Database, since time.Time, until time.Time) (*History, error) {
	held := &History{}
	for _, record := range records {
		if record.Kind() != models.EnginePostgres || !record.Available() {
			continue
		}
		base := database.DB.WithContext(ctx).Model(&models.PostgresStatistics{}).Where("database_id = ? AND cluster_id = ? AND database_oid = ? AND version = ?", record.ID, record.ClusterID, record.BackendOID, SampleVersion)
		var latest models.PostgresStatistics
		err := base.Order("collected_at DESC").First(&latest).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if latest.ID != 0 {
			held.Latest = append(held.Latest, latest)
			if held.LastCollectedAt == nil || latest.CollectedAt.After(*held.LastCollectedAt) {
				moment := latest.CollectedAt
				held.LastCollectedAt = &moment
			}
		}
		var earliest models.PostgresStatistics
		err = database.DB.WithContext(ctx).Where("database_id = ? AND cluster_id = ? AND database_oid = ? AND version = ? AND collected_at >= ? AND collected_at <= ?", record.ID, record.ClusterID, record.BackendOID, SampleVersion, since, until).Order("collected_at ASC").First(&earliest).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if earliest.ID != 0 && (held.FirstCollectedAt == nil || earliest.CollectedAt.Before(*held.FirstCollectedAt)) {
			moment := earliest.CollectedAt
			held.FirstCollectedAt = &moment
		}
		var bounds struct {
			Intervals       int64
			Discontinuities int64
			Committed       int64
			RolledBack      int64
			Deadlocks       int64
		}
		err = database.DB.WithContext(ctx).Model(&models.PostgresStatistics{}).
			Where("database_id = ? AND cluster_id = ? AND database_oid = ? AND version = ? AND collected_at >= ? AND collected_at <= ?", record.ID, record.ClusterID, record.BackendOID, SampleVersion, since, until).
			Select("SUM(CASE WHEN statements_continuous AND interval_started_at >= ? THEN 1 ELSE 0 END) AS intervals, SUM(CASE WHEN NOT statements_continuous THEN 1 ELSE 0 END) AS discontinuities, SUM(CASE WHEN continuous AND interval_started_at >= ? THEN committed_delta ELSE 0 END) AS committed, SUM(CASE WHEN continuous AND interval_started_at >= ? THEN rolled_back_delta ELSE 0 END) AS rolled_back, SUM(CASE WHEN continuous AND interval_started_at >= ? THEN deadlocks_delta ELSE 0 END) AS deadlocks", since, since, since, since).Scan(&bounds).Error
		if err != nil {
			return nil, err
		}
		held.Intervals += bounds.Intervals
		held.Discontinuities += bounds.Discontinuities
		held.Committed += bounds.Committed
		held.RolledBack += bounds.RolledBack
		held.Deadlocks += bounds.Deadlocks
		var statementBaselines int64
		err = database.DB.WithContext(ctx).Model(&models.PostgresStatementStatistics{}).Where("database_id = ? AND cluster_id = ? AND database_oid = ? AND version = ? AND top_level = ? AND continuous = ? AND collected_at >= ? AND collected_at <= ?", record.ID, record.ClusterID, record.BackendOID, SampleVersion, true, false, since, until).Count(&statementBaselines).Error
		if err != nil {
			return nil, err
		}
		held.Discontinuities += statementBaselines
		var queries []QuerySummary
		err = database.DB.WithContext(ctx).Model(&models.PostgresStatementStatistics{}).
			Where("database_id = ? AND cluster_id = ? AND database_oid = ? AND version = ? AND top_level = ? AND continuous = ? AND interval_started_at >= ? AND collected_at <= ?", record.ID, record.ClusterID, record.BackendOID, SampleVersion, true, true, since, until).
			Select("database_id, cluster_id, database_oid, native_oid, query_id, top_level, statistics_since, statement, SUM(calls_delta) AS calls, SUM(rows_delta) AS rows, SUM(execution_delta_ms) AS execution_ms, SUM(planning_delta_ms) AS planning_ms, MIN(minimum_execution_ms) AS minimum_execution_ms, MAX(maximum_execution_ms) AS maximum_execution_ms, SUM(shared_blocks_hit_delta) AS shared_blocks_hit, SUM(shared_blocks_read_delta) AS shared_blocks_read, SUM(local_blocks_hit_delta) AS local_blocks_hit, SUM(local_blocks_read_delta) AS local_blocks_read, SUM(temporary_blocks_read_delta) AS temporary_blocks_read, SUM(temporary_blocks_written_delta) AS temporary_blocks_written").
			Group("database_id, cluster_id, database_oid, native_oid, query_id, top_level, statistics_since, statement").Having("SUM(calls_delta) > 0").Order("execution_ms DESC").Scan(&queries).Error
		if err != nil {
			return nil, err
		}
		for index := range queries {
			queries[index].DatabaseName = record.Name
			held.Calls += queries[index].Calls
			held.ExecutionMs += queries[index].ExecutionMs
			held.Rows += queries[index].Rows
			held.SharedBlocksHit += queries[index].SharedBlocksHit
			held.SharedBlocksRead += queries[index].SharedBlocksRead
		}
		held.Queries = append(held.Queries, queries...)
	}
	return held, nil
}
