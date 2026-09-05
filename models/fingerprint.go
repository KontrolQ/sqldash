package models

import "gorm.io/gorm"

type Fingerprint struct {
	gorm.Model
	DatabaseName string `gorm:"index:idx_fingerprint_database_digest,unique;not null"`
	Digest       string `gorm:"index:idx_fingerprint_database_digest,unique;not null"`
	Normalised   string `gorm:"not null"`
	Example      string
}
