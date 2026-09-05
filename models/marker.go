package models

type Marker struct {
	Name  string `gorm:"primarykey"`
	Value int64  `gorm:"not null"`
}
