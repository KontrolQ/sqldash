package analytics

import (
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/utils/logger"

	"gorm.io/gorm/clause"
)

type slot struct {
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
				databaseName: statement.DatabaseName,
				digest:       statement.Digest,
				resolution:   resolution,
				bucketStart:  truncateTo(statement.OccurredAt, resolution),
			}

			held, seen := gathered[key]
			if !seen {
				held = &models.Rollup{
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

			if statement.Failed {
				held.Failures++
			}

			if statement.RowsWritten > 0 {
				held.Writes++
			}

			histograms[key].Add(statement.DurationMs)
		}
	}

	for key, held := range gathered {
		if applyError := apply(held, *histograms[key]); applyError != nil {
			logger.Errorf(LogPrefix, FoldFailedLog, applyError)
			return
		}
	}

	setMarker(FoldedMarker, int64(statements[len(statements)-1].ID))
}

func apply(fresh *models.Rollup, histogram Histogram) error {
	existing := &models.Rollup{}

	findError := database.DB.
		Where("database_name = ? AND digest = ? AND resolution = ? AND bucket_start = ?",
			fresh.DatabaseName, fresh.Digest, fresh.Resolution, fresh.BucketStart).
		First(existing).Error

	if findError == nil {
		existing.Count += fresh.Count
		existing.Failures += fresh.Failures
		existing.Writes += fresh.Writes
		existing.TotalDuration += fresh.TotalDuration
		existing.RowsRead += fresh.RowsRead
		existing.RowsWritten += fresh.RowsWritten
		existing.RowsReturned += fresh.RowsReturned

		merged := DecodeHistogram(existing.Histogram)
		merged.Merge(histogram)
		existing.Histogram = merged.Encode()

		return database.DB.Save(existing).Error
	}

	fresh.Histogram = histogram.Encode()

	return database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(fresh).Error
}

func Prune() {
	now := time.Now()

	cuts := []struct {
		query string
		value any
	}{
		{"occurred_at < ?", now.Add(-RawRetention)},
	}

	for _, cut := range cuts {
		if deleteError := database.DB.Where(cut.query, cut.value).Delete(&models.Statement{}).Error; deleteError != nil {
			logger.Warnf(LogPrefix, PruneFailedLog, deleteError)
		}
	}

	retentions := map[models.Resolution]time.Duration{
		models.ResolutionMinute: MinuteRetention,
		models.ResolutionHour:   HourRetention,
		models.ResolutionDay:    DayRetention,
	}

	for resolution, keep := range retentions {
		deleteError := database.DB.
			Where("resolution = ? AND bucket_start < ?", resolution, now.Add(-keep)).
			Delete(&models.Rollup{}).Error

		if deleteError != nil {
			logger.Warnf(LogPrefix, PruneFailedLog, deleteError)
		}
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

func setMarker(name string, value int64) {
	database.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&models.Marker{Name: name, Value: value})
}
