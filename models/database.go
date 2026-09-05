package models

import "gorm.io/gorm"

type Database struct {
	gorm.Model
	Name      string `gorm:"uniqueIndex;not null"`
	Protected bool   `gorm:"not null;default:false"`
	Notes     string
}
