package telemetry

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sqldash/database"
	"sqldash/models"
	"sqldash/utils/logger"
	"sync"
	"time"
)

var queue = make(chan Observation, QueueDepth)
var startGuard sync.Once

func Start() {
	startGuard.Do(func() { go drain() })
}

func Record(observation Observation) {
	var record models.Database
	if findError := database.DB.Where("name = ? AND engine = ? AND state = ?", observation.DatabaseName, models.EngineLibSQL, models.DatabaseReady).First(&record).Error; findError != nil {
		return
	}
	observation.DatabaseID = record.ID
	select {
	case queue <- observation:
	default:
		logger.Warnf(LogPrefix, QueueFullLog)
	}
}

func drain() {
	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()
	batch := make([]Observation, 0, BatchSize)
	for {
		select {
		case observation := <-queue:
			batch = append(batch, observation)
			if len(batch) >= BatchSize {
				write(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				write(batch)
				batch = batch[:0]
			}
		}
	}
}

func write(batch []Observation) {
	writeError := database.DB.Transaction(func(transaction *gorm.DB) error {
		statements := make([]models.Statement, 0, len(batch))
		fingerprints := make([]models.Fingerprint, 0, len(batch))
		seen := make(map[string]bool, len(batch))
		for _, observation := range batch {
			var record models.Database
			findError := transaction.Where("id = ? AND name = ? AND engine = ? AND state = ?", observation.DatabaseID, observation.DatabaseName, models.EngineLibSQL, models.DatabaseReady).First(&record).Error
			if findError != nil {
				continue
			}
			normalised := Normalise(observation.SQL)
			digest := Digest(normalised)
			statements = append(statements, models.Statement{
				DatabaseID: record.ID, DatabaseName: observation.DatabaseName, Digest: digest,
				DurationMs: observation.DurationMs, RowsRead: observation.RowsRead, RowsWritten: observation.RowsWritten,
				RowsReturned: observation.RowsReturned, BytesIn: observation.BytesIn, BytesOut: observation.BytesOut,
				Failed: observation.Failed, OccurredAt: observation.OccurredAt,
			})
			key := observation.DatabaseName + digest
			if seen[key] {
				continue
			}
			seen[key] = true
			fingerprints = append(fingerprints, models.Fingerprint{DatabaseID: record.ID, DatabaseName: observation.DatabaseName, Digest: digest, Normalised: normalised, Example: observation.SQL})
		}
		if len(fingerprints) > 0 {
			if createError := transaction.Clauses(clause.OnConflict{DoNothing: true}).Create(&fingerprints).Error; createError != nil {
				return createError
			}
		}
		if len(statements) > 0 {
			return transaction.Create(&statements).Error
		}
		return nil
	})
	if writeError != nil {
		logger.Errorf(LogPrefix, WriteFailedLog, len(batch), writeError)
	}
}
