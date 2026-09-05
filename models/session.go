package models

import "time"

type Session struct {
	Key       string `gorm:"primarykey"`
	Payload   []byte `gorm:"not null"`
	ExpiresAt *time.Time
}
