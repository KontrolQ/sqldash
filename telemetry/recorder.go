package telemetry

import (
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/utils/logger"

	"gorm.io/gorm/clause"
)

var queue = make(chan Observation, QueueDepth)

func Start() {
	go drain()
}

func Record(observation Observation) {
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
	statements := make([]models.Statement, 0, len(batch))
	fingerprints := make([]models.Fingerprint, 0, len(batch))
	seen := make(map[string]bool, len(batch))

	for _, observation := range batch {
		normalised := Normalise(observation.SQL)
		digest := Digest(normalised)

		statements = append(statements, models.Statement{
			DatabaseName: observation.DatabaseName,
			Digest:       digest,
			DurationMs:   observation.DurationMs,
			RowsRead:     observation.RowsRead,
			RowsWritten:  observation.RowsWritten,
			RowsReturned: observation.RowsReturned,
			BytesIn:      observation.BytesIn,
			BytesOut:     observation.BytesOut,
			Failed:       observation.Failed,
			OccurredAt:   observation.OccurredAt,
		})

		key := observation.DatabaseName + digest
		if seen[key] {
			continue
		}

		seen[key] = true
		fingerprints = append(fingerprints, models.Fingerprint{
			DatabaseName: observation.DatabaseName,
			Digest:       digest,
			Normalised:   normalised,
			Example:      observation.SQL,
		})
	}

	if createError := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&fingerprints).Error; createError != nil {
		logger.Warnf(LogPrefix, FingerprintLog, fingerprints[0].DatabaseName, createError)
	}

	if createError := database.DB.Create(&statements).Error; createError != nil {
		logger.Errorf(LogPrefix, WriteFailedLog, len(statements), createError)
	}
}
