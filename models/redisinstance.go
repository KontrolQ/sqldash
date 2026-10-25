package models

import (
	"time"

	"gorm.io/gorm"
)

type RedisInstance struct {
	gorm.Model
	DatabaseID     uint   `gorm:"uniqueIndex;not null"`
	Identity       string `gorm:"uniqueIndex;not null"`
	Generation     uint64 `gorm:"not null;default:1"`
	State          string `gorm:"index;not null;default:pending"`
	Published      bool   `gorm:"not null;default:false"`
	MemoryLimit    int64  `gorm:"not null;default:268435456"`
	EvictionPolicy string `gorm:"not null;default:noeviction"`
	Persistence    string `gorm:"not null;default:aof-rdb"`
	Fsync          string `gorm:"not null;default:everysec"`
	ProcessID      int
	RunID          string
	Failure        string
	ReadyAt        *time.Time
	StoppedAt      *time.Time
}
