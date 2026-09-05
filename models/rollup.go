package models

import "time"

type Resolution string

const (
	ResolutionMinute Resolution = "minute"
	ResolutionHour   Resolution = "hour"
	ResolutionDay    Resolution = "day"
)

type Rollup struct {
	ID            uint       `gorm:"primarykey"`
	DatabaseName  string     `gorm:"index:idx_rollup_slot,unique;not null"`
	Digest        string     `gorm:"index:idx_rollup_slot,unique;not null"`
	Resolution    Resolution `gorm:"index:idx_rollup_slot,unique;not null"`
	BucketStart   time.Time  `gorm:"index:idx_rollup_slot,unique;index;not null"`
	Count         int64      `gorm:"not null"`
	Failures      int64      `gorm:"not null"`
	Writes        int64      `gorm:"not null;default:0"`
	TotalDuration float64    `gorm:"not null"`
	RowsRead      int64      `gorm:"not null"`
	RowsWritten   int64      `gorm:"not null"`
	RowsReturned  int64      `gorm:"not null"`
	BytesIn       int64      `gorm:"not null;default:0"`
	BytesOut      int64      `gorm:"not null;default:0"`
	Histogram     string     `gorm:"not null"`
}
