package models

import "gorm.io/gorm"

type Account struct {
	gorm.Model
	Username           string `gorm:"uniqueIndex;not null"`
	DisplayName        string `gorm:"type:text"`
	Email              string `gorm:"uniqueIndex;not null"`
	PasswordHash       string `gorm:"not null"`
	ProfilePicture     string `gorm:"type:text"`
	ProfilePictureType string `gorm:"type:text"`
	Active             bool   `gorm:"not null;default:true"`
	AllDatabases       bool   `gorm:"not null;default:false"`
	SessionVersion     uint   `gorm:"not null;default:0"`
}
