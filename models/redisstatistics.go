package models

import "time"

type RedisStatistics struct {
	ID         uint      `gorm:"primaryKey"`
	DatabaseID uint      `gorm:"index:redis_statistics_window;not null"`
	RecordedAt time.Time `gorm:"index:redis_statistics_window;not null"`
	Identity   string    `gorm:"not null"`
	RunID      string    `gorm:"not null"`
	Payload    string    `gorm:"not null"`
}
