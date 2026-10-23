package models

import (
	"gorm.io/gorm"
	"time"
)

type PostgresConnection struct {
	gorm.Model
	DatabaseID        uint `gorm:"index;not null;index:idx_postgres_primary,unique,where:primary_connection = 1 AND deleted_at IS NULL"`
	PrimaryConnection bool `gorm:"not null;default:false"`
	PasswordReference string
	AccountID         uint       `gorm:"index;not null"`
	Label             string     `gorm:"not null"`
	Username          string     `gorm:"uniqueIndex;not null"`
	Scope             TokenScope `gorm:"not null"`
	State             string     `gorm:"index;not null;default:pending"`
	Generation        uint64     `gorm:"not null;default:1"`
	PendingSecret     string
	ExpiresAt         *time.Time
	RevokedAt         *time.Time
	LastUsedAt        *time.Time
}

func (self *PostgresConnection) Usable(now time.Time) bool {
	return self.State == DatabaseReady && self.RevokedAt == nil && (self.ExpiresAt == nil || now.Before(*self.ExpiresAt))
}
