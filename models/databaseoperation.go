package models

import (
	"gorm.io/gorm"
	"time"
)

type DatabaseOperation struct {
	gorm.Model
	DatabaseID       uint   `gorm:"index;not null"`
	SourceDatabaseID uint   `gorm:"index;not null;default:0"`
	ConnectionID     uint   `gorm:"index;not null;default:0"`
	AccountID        uint   `gorm:"index;not null"`
	Engine           string `gorm:"index;not null"`
	Kind             string `gorm:"index;not null"`
	State            string `gorm:"index;not null;default:pending"`
	Payload          string
	Failure          string
	Attempts         uint `gorm:"not null;default:0"`
	LeaseUntil       *time.Time
}
