package models

import (
	"gorm.io/gorm"
	"time"
)

type OwnershipTransfer struct {
	gorm.Model
	OwnerAccountID     uint      `gorm:"not null"`
	RecipientAccountID uint      `gorm:"index;not null"`
	Status             string    `gorm:"index;not null"`
	ExpiresAt          time.Time `gorm:"not null"`
}
