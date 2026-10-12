package models

type DatabaseShare struct {
	DatabaseID  uint `gorm:"primaryKey"`
	AccountID   uint `gorm:"primaryKey"`
	GrantedByID uint
}
