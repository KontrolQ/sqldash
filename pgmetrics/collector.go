package pgmetrics

import (
	"context"
	"sync"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/pgcluster/control"
	"sqldash/utils/logger"
)

var startOnce sync.Once
var sampleLock sync.Mutex

func Start(ctx context.Context) {
	startOnce.Do(func() { go SampleLoop(ctx) })
}

func SampleLoop(ctx context.Context) {
	ticker := time.NewTicker(SampleInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		var records []models.Database
		if err := database.DB.WithContext(ctx).Where("engine = ? AND state = ?", models.EnginePostgres, models.DatabaseReady).Find(&records).Error; err != nil {
			if ctx.Err() == nil {
				logger.Errorf(LogPrefix, RegistrationsUnavailable)
			}
		} else {
			for _, record := range records {
				if err := SampleDatabase(ctx, &record); err != nil && ctx.Err() == nil {
					logger.Errorf(LogPrefix, SamplingFailed, record.ID)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func DatabaseStatistics(ctx context.Context, record *models.Database) (*control.StatisticsRecord, error) {
	bounded, cancel := context.WithTimeout(ctx, SampleTimeout)
	defer cancel()
	return control.Statistics(bounded, record)
}

func SampleDatabase(ctx context.Context, record *models.Database) error {
	if record == nil || record.Kind() != models.EnginePostgres || record.ID == 0 || record.BackendOID == 0 || record.ClusterID == "" {
		return IdentityMismatch
	}
	sampleLock.Lock()
	defer sampleLock.Unlock()
	bounded, cancel := context.WithTimeout(ctx, SampleTimeout)
	defer cancel()
	statistics, err := control.Statistics(bounded, record)
	if err != nil {
		return err
	}
	statements, err := control.Statements(bounded, record)
	if err != nil {
		return err
	}
	confirmed, err := control.Statistics(bounded, record)
	if err != nil {
		return err
	}
	if statistics.ClusterID != record.ClusterID || statistics.DatabaseOID != record.BackendOID ||
		confirmed.ClusterID != statistics.ClusterID || confirmed.DatabaseOID != statistics.DatabaseOID ||
		!confirmed.PostmasterStartedAt.Equal(statistics.PostmasterStartedAt) ||
		!sameTime(confirmed.ResetAt, statistics.ResetAt) || !sameTime(confirmed.StatementResetAt, statistics.StatementResetAt) ||
		confirmed.StatementsDeallocated != statistics.StatementsDeallocated || confirmed.StatementsAvailable != statistics.StatementsAvailable {
		return IdentityMismatch
	}
	return persist(bounded, record, statistics, statements, time.Now().UTC())
}

func sameTime(left *time.Time, right *time.Time) bool {
	return left == nil && right == nil || left != nil && right != nil && left.Equal(*right)
}
