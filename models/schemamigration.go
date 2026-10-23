package models

import "time"

type SchemaMigration struct {
	Name      string    `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}
