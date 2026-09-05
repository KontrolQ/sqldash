package models

import "gorm.io/gorm"

type Snippet struct {
	gorm.Model
	Database  string `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	Statement string `gorm:"not null"`
}
