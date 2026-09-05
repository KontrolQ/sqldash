package models

import (
	"time"

	"gorm.io/gorm"
)

type TokenScope string

const (
	ScopeReadOnly  TokenScope = "read-only"
	ScopeReadWrite TokenScope = "read-write"
)

type Token struct {
	gorm.Model
	DatabaseName string     `gorm:"index;not null"`
	Label        string     `gorm:"not null"`
	Identifier   string     `gorm:"uniqueIndex;not null"`
	Scope        TokenScope `gorm:"not null"`
	ExpiresAt    *time.Time
	RevokedAt    *time.Time
	LastUsedAt   *time.Time
}

func (self *Token) Usable(now time.Time) bool {
	if self.RevokedAt != nil {
		return false
	}

	if self.ExpiresAt != nil && now.After(*self.ExpiresAt) {
		return false
	}

	return true
}
