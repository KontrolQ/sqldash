package models

import "gorm.io/gorm"

type Fingerprint struct {
	gorm.Model
	DatabaseName string `gorm:"index:idx_fingerprint_database_digest,unique;not null"`
	DatabaseID   uint   `gorm:"index;not null;default:0"`
	Digest       string `gorm:"index:idx_fingerprint_database_digest,unique;not null"`
	Normalised   string `gorm:"not null"`
	Example      string
}
