package analytics

import (
	"time"

	"sqldash/database"
	"sqldash/models"
)

type Window struct {
	Label      string
	Since      time.Duration
	Resolution models.Resolution
}

type Summary struct {
	Count         int64
	Failures      int64
	Writes        int64
	TotalDuration float64
	RowsRead      int64
	RowsWritten   int64
	RowsReturned  int64
	Histogram     Histogram
}

type QuerySummary struct {
	Summary
	Digest     string
	Normalised string
	Example    string
}

func (self Summary) Average() float64 {
	if self.Count == 0 {
		return 0
	}

	return self.TotalDuration / float64(self.Count)
}

func Windows() []Window {
	return []Window{
		{Label: "24 hours", Since: 24 * time.Hour, Resolution: models.ResolutionMinute},
		{Label: "7 days", Since: 7 * 24 * time.Hour, Resolution: models.ResolutionHour},
		{Label: "30 days", Since: 30 * 24 * time.Hour, Resolution: models.ResolutionHour},
		{Label: "90 days", Since: 90 * 24 * time.Hour, Resolution: models.ResolutionDay},
	}
}

func WindowNamed(label string) Window {
	for _, window := range Windows() {
		if window.Label == label {
			return window
		}
	}

	return Windows()[0]
}

func rollupsIn(databaseName string, window Window) ([]models.Rollup, error) {
	ends := time.Now()

	return rollupsBetween(databaseName, window.Resolution, ends.Add(-window.Since), ends)
}

func rollupsBetween(databaseName string, resolution models.Resolution, from time.Time, to time.Time) ([]models.Rollup, error) {
	records := make([]models.Rollup, 0)

	query := database.DB.
		Where("resolution = ? AND bucket_start >= ? AND bucket_start < ?", resolution, from, to)

	if databaseName != "" {
		query = query.Where("database_name = ?", databaseName)
	}

	if findError := query.Find(&records).Error; findError != nil {
		return nil, findError
	}

	return records, nil
}

func Overall(databaseName string, window Window) (*Summary, error) {
	records, findError := rollupsIn(databaseName, window)
	if findError != nil {
		return nil, findError
	}

	summary := &Summary{}

	for _, record := range records {
		summary.absorb(record)
	}

	return summary, nil
}

func Preceding(databaseName string, window Window) (*Summary, error) {
	ends := time.Now().Add(-window.Since)

	records, findError := rollupsBetween(databaseName, window.Resolution, ends.Add(-window.Since), ends)
	if findError != nil {
		return nil, findError
	}

	summary := &Summary{}

	for _, record := range records {
		summary.absorb(record)
	}

	return summary, nil
}

func TopQueries(databaseName string, window Window, limit int) ([]QuerySummary, error) {
	summaries, findError := EveryQuery(databaseName, window)
	if findError != nil {
		return nil, findError
	}

	if limit > 0 && len(summaries) > limit {
		summaries = summaries[:limit]
	}

	nameThem(databaseName, summaries)

	return summaries, nil
}

func EveryQuery(databaseName string, window Window) ([]QuerySummary, error) {
	records, findError := rollupsIn(databaseName, window)
	if findError != nil {
		return nil, findError
	}

	gathered := make(map[string]*QuerySummary)

	for _, record := range records {
		held, seen := gathered[record.Digest]
		if !seen {
			held = &QuerySummary{Digest: record.Digest}
			gathered[record.Digest] = held
		}

		held.absorb(record)
	}

	summaries := make([]QuerySummary, 0, len(gathered))
	for _, held := range gathered {
		summaries = append(summaries, *held)
	}

	sortByTotalDuration(summaries)

	return summaries, nil
}

func Name(databaseName string, summaries []QuerySummary) {
	nameThem(databaseName, summaries)
}

func (self *Summary) absorb(record models.Rollup) {
	self.Count += record.Count
	self.Failures += record.Failures
	self.Writes += record.Writes
	self.TotalDuration += record.TotalDuration
	self.RowsRead += record.RowsRead
	self.RowsWritten += record.RowsWritten
	self.RowsReturned += record.RowsReturned
	self.Histogram.Merge(DecodeHistogram(record.Histogram))
}

func sortByTotalDuration(summaries []QuerySummary) {
	for outer := 1; outer < len(summaries); outer++ {
		held := summaries[outer]
		inner := outer - 1

		for inner >= 0 && summaries[inner].TotalDuration < held.TotalDuration {
			summaries[inner+1] = summaries[inner]
			inner--
		}

		summaries[inner+1] = held
	}
}

func nameThem(databaseName string, summaries []QuerySummary) {
	digests := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		digests = append(digests, summary.Digest)
	}

	fingerprints := make([]models.Fingerprint, 0)

	query := database.DB.Where("digest IN ?", digests)
	if databaseName != "" {
		query = query.Where("database_name = ?", databaseName)
	}

	if findError := query.Find(&fingerprints).Error; findError != nil {
		return
	}

	byDigest := make(map[string]models.Fingerprint, len(fingerprints))
	for _, fingerprint := range fingerprints {
		byDigest[fingerprint.Digest] = fingerprint
	}

	for index := range summaries {
		if held, seen := byDigest[summaries[index].Digest]; seen {
			summaries[index].Normalised = held.Normalised
			summaries[index].Example = held.Example
		}
	}
}
