package models

type Site struct {
	ID                 uint `gorm:"primaryKey"`
	OwnerAccountID     uint `gorm:"not null"`
	LegacyRolesRemoved bool `gorm:"not null;default:false"`
}
