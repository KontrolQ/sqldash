package models

import "time"

type Statement struct {
	ID           uint      `gorm:"primarykey"`
	DatabaseName string    `gorm:"index;not null"`
	Digest       string    `gorm:"index;not null"`
	DurationMs   float64   `gorm:"not null"`
	RowsRead     int64     `gorm:"not null"`
	RowsWritten  int64     `gorm:"not null"`
	RowsReturned int64     `gorm:"not null"`
	Failed       bool      `gorm:"not null;default:false"`
	OccurredAt   time.Time `gorm:"index;not null"`
}
