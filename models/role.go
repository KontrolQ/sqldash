package models

import "gorm.io/gorm"

type Role struct {
	gorm.Model
	Name     string `gorm:"uniqueIndex;not null"`
	Color    string `gorm:"not null"`
	Position int    `gorm:"not null"`
	Kind     string
}

type RolePermission struct {
	RoleID     uint   `gorm:"primaryKey"`
	Permission string `gorm:"primaryKey"`
}

type AccountRole struct {
	AccountID uint `gorm:"primaryKey"`
	RoleID    uint `gorm:"primaryKey"`
}
