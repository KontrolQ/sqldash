package models

import "gorm.io/gorm"

type AccountPicture struct {
	gorm.Model
	AccountID          uint   `gorm:"index;not null"`
	ProfilePicture     string `gorm:"type:text;not null"`
	ProfilePictureType string `gorm:"type:text;not null"`
}
