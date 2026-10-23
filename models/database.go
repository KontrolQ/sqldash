package models

import "gorm.io/gorm"

type Database struct {
	gorm.Model
	Name           string `gorm:"uniqueIndex;not null"`
	Protected      bool   `gorm:"not null;default:false"`
	Notes          string
	OwnerAccountID uint   `gorm:"index;not null;default:0"`
	Engine         string `gorm:"index;not null;default:libsql"`
	State          string `gorm:"index;not null;default:ready"`
	BackendOID     uint32 `gorm:"column:backend_oid;index;not null;default:0"`
	ClusterID      string
	PolicyVersion  uint64 `gorm:"not null;default:1"`
}
