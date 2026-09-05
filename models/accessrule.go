package models

import "gorm.io/gorm"

type AccessRule struct {
	gorm.Model
	DatabaseName string `gorm:"index;not null"`
	Network      string `gorm:"not null"`
	Note         string
}
