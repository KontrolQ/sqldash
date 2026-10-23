package models

import (
	"time"

	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	OccurredAt time.Time `gorm:"index;not null"`
	Database   string    `gorm:"index"`
	DatabaseID uint      `gorm:"index;not null;default:0"`
	Engine     string    `gorm:"index"`
	Action     string    `gorm:"index;not null"`
	Detail     string
	Actor      string
}
