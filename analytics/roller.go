package analytics

import (
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/utils/logger"

	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sync"
)

var foldLock sync.Mutex

type slot struct {
	databaseID   uint
	databaseName string
	digest       string
	resolution   models.Resolution
	bucketStart  time.Time
}

func Start() {
	go func() {
		rolling := time.NewTicker(RollInterval)
		pruning := time.NewTicker(PruneInterval)

		defer rolling.Stop()
		defer pruning.Stop()

		Fold()
		Prune()

		for {
			select {
			case <-rolling.C:
				Fold()
			case <-pruning.C:
				Prune()
			}
		}
	}()
}

func Fold() {
	foldLock.Lock()
	defer foldLock.Unlock()
	watermark := markerValue(FoldedMarker)

	statements := make([]models.Statement, 0, FoldBatch)

	findError := database.DB.
		Where("id > ?", watermark).
		Order("id asc").
		Limit(FoldBatch).
		Find(&statements).Error

	if findError != nil {
		logger.Errorf(LogPrefix, FoldFailedLog, findError)
		return
	}

	if len(statements) == 0 {
		return
	}

	gathered := make(map[slot]*models.Rollup)
	histograms := make(map[slot]*Histogram)

	for _, statement := range statements {
		for _, resolution := range []models.Resolution{models.ResolutionMinute, models.ResolutionHour, models.ResolutionDay} {
			key := slot{
				databaseID:   statement.DatabaseID,
				databaseName: statement.DatabaseName,
				digest:       statement.Digest,
				resolution:   resolution,
				bucketStart:  truncateTo(statement.OccurredAt, resolution),
			}

			held, seen := gathered[key]
			if !seen {
				held = &models.Rollup{
					DatabaseID:   key.databaseID,
					DatabaseName: key.databaseName,
					Digest:       key.digest,
					Resolution:   key.resolution,
					BucketStart:  key.bucketStart,
				}

				gathered[key] = held
				histograms[key] = &Histogram{}
			}

			held.Count++
			held.TotalDuration += statement.DurationMs
			held.RowsRead += statement.RowsRead
			held.RowsWritten += statement.RowsWritten
			held.RowsReturned += statement.RowsReturned
			held.BytesIn += statement.BytesIn
			held.BytesOut += statement.BytesOut

			if statement.Failed {
				held.Failures++
			}

			if statement.RowsWritten > 0 {
				held.Writes++
			}

			histograms[key].Add(statement.DurationMs)
		}
	}

	foldError := database.DB.Transaction(func(store *gorm.DB) error {
		var marker models.Marker
		if err := store.Where("name = ?", FoldedMarker).First(&marker).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if marker.Value != watermark {
			return nil
		}
		for key, held := range gathered {
			if err := apply(store, held, *histograms[key]); err != nil {
				return err
			}
		}
		return store.Clauses(clause.OnConflict{UpdateAll: true}).Create(&models.Marker{Name: FoldedMarker, Value: int64(statements[len(statements)-1].ID)}).Error
	})
	if foldError != nil {
		logger.Errorf(LogPrefix, FoldFailedLog, foldError)
	}
}

func apply(store *gorm.DB, fresh *models.Rollup, histogram Histogram) error {
	var registration models.Database
	lookupError := store.Where("id = ? AND name = ? AND engine IN ? AND state IN ?", fresh.DatabaseID, fresh.DatabaseName, []string{"", models.EngineLibSQL}, []string{"", models.DatabaseReady}).First(&registration).Error
	if errors.Is(lookupError, gorm.ErrRecordNotFound) {
		return nil
	}
	if lookupError != nil {
		return lookupError
	}
	existing := &models.Rollup{}

	findError := store.
		Where("database_id = ? AND database_name = ? AND digest = ? AND resolution = ? AND bucket_start = ?",
			fresh.DatabaseID, fresh.DatabaseName, fresh.Digest, fresh.Resolution, fresh.BucketStart).
		First(existing).Error

	if findError == nil {
		existing.Count += fresh.Count
		existing.Failures += fresh.Failures
		existing.Writes += fresh.Writes
		existing.TotalDuration += fresh.TotalDuration
		existing.RowsRead += fresh.RowsRead
		existing.RowsWritten += fresh.RowsWritten
		existing.RowsReturned += fresh.RowsReturned
		existing.BytesIn += fresh.BytesIn
		existing.BytesOut += fresh.BytesOut

		merged := DecodeHistogram(existing.Histogram)
		merged.Merge(histogram)
		existing.Histogram = merged.Encode()

		return store.Save(existing).Error
	}

	if !errors.Is(findError, gorm.ErrRecordNotFound) {
		return findError
	}
	fresh.Histogram = histogram.Encode()
	return store.Create(fresh).Error
}

func Prune() {
	if config.History.Retention == 0 {
		return
	}
	cutoff := time.Now().Add(-config.History.Retention)
	if err := database.DB.Where("occurred_at < ?", cutoff).Delete(&models.Statement{}).Error; err != nil {
		logger.Warnf(LogPrefix, PruneFailedLog, err)
	}
	if err := database.DB.Where("bucket_start < ?", cutoff).Delete(&models.Rollup{}).Error; err != nil {
		logger.Warnf(LogPrefix, PruneFailedLog, err)
	}
}

func truncateTo(moment time.Time, resolution models.Resolution) time.Time {
	moment = moment.UTC()

	switch resolution {
	case models.ResolutionMinute:
		return moment.Truncate(time.Minute)
	case models.ResolutionHour:
		return moment.Truncate(time.Hour)
	default:
		return time.Date(moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func markerValue(name string) int64 {
	held := &models.Marker{}

	if findError := database.DB.Where("name = ?", name).First(held).Error; findError != nil {
		return 0
	}

	return held.Value
}
