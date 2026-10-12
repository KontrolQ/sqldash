package models

import (
	"gorm.io/gorm"
	"time"
)

type AccountLink struct {
	gorm.Model
	Kind         string `gorm:"index;not null"`
	Digest       string `gorm:"uniqueIndex;not null"`
	Email        string `gorm:"index;not null"`
	IssuerID     uint   `gorm:"not null"`
	AccountID    uint
	RoleIDs      string `gorm:"type:text"`
	DatabaseIDs  string `gorm:"type:text"`
	AllDatabases bool
	ExpiresAt    time.Time `gorm:"not null"`
	UsedAt       *time.Time
	RevokedAt    *time.Time
}
