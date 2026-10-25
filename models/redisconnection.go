package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type RedisConnection struct {
	gorm.Model
	DatabaseID        uint   `gorm:"index;not null;index:idx_redis_primary,unique,where:primary_connection = 1 AND deleted_at IS NULL"`
	InstanceIdentity  string `gorm:"not null"`
	AccountID         uint   `gorm:"index;not null"`
	Username          string `gorm:"uniqueIndex;not null"`
	Label             string
	Scope             TokenScope `gorm:"not null"`
	PrimaryConnection bool       `gorm:"index;not null;default:false"`
	Subscribe         bool       `gorm:"not null;default:false"`
	Publish           bool       `gorm:"not null;default:false"`
	Generation        uint64     `gorm:"not null;default:1"`
	PasswordDigest    string
	PendingSecret     string
	State             string `gorm:"index;not null;default:pending"`
	ExpiresAt         *time.Time
	RevokedAt         *time.Time
}

func (self *RedisConnection) Usable(now time.Time) bool {
	return self != nil && !self.DeletedAt.Valid && self.State == DatabaseReady && self.Generation > 0 && self.InstanceIdentity != "" && PublicRedisUsername(self.Username) && (self.Scope == ScopeReadOnly || self.Scope == ScopeReadWrite) && self.RevokedAt == nil && (self.ExpiresAt == nil || now.Before(*self.ExpiresAt))
}

func ValidRedisUsername(username string) bool {
	if len(username) < 1 || len(username) > 128 || username == "default" {
		return false
	}
	for _, character := range username {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func PublicRedisUsername(username string) bool {
	return ValidRedisUsername(username) && !strings.Contains(username, "_system_")
}
