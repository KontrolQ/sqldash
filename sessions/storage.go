package sessions

import (
	"context"
	"time"

	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type store struct{}

func (self store) Get(key string) ([]byte, error) {
	return self.GetWithContext(context.Background(), key)
}

func (self store) GetWithContext(_ context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}

	held := &models.Session{}

	findError := database.DB.Where("key = ?", key).First(held).Error
	if findError == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if findError != nil {
		return nil, findError
	}

	if held.ExpiresAt != nil && time.Now().After(*held.ExpiresAt) {
		database.DB.Delete(held)
		return nil, nil
	}

	return held.Payload, nil
}

func (self store) Set(key string, payload []byte, lifetime time.Duration) error {
	return self.SetWithContext(context.Background(), key, payload, lifetime)
}

func (self store) SetWithContext(_ context.Context, key string, payload []byte, lifetime time.Duration) error {
	if key == "" || len(payload) == 0 {
		return nil
	}

	held := &models.Session{Key: key, Payload: payload}

	if lifetime > 0 {
		expires := time.Now().Add(lifetime)
		held.ExpiresAt = &expires
	}

	return database.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(held).Error
}

func (self store) Delete(key string) error {
	return self.DeleteWithContext(context.Background(), key)
}

func (self store) DeleteWithContext(_ context.Context, key string) error {
	if key == "" {
		return nil
	}

	return database.DB.Where("key = ?", key).Delete(&models.Session{}).Error
}

func (self store) Reset() error {
	return self.ResetWithContext(context.Background())
}

func (self store) ResetWithContext(_ context.Context) error {
	return database.DB.Where("1 = 1").Delete(&models.Session{}).Error
}

func (self store) Close() error {
	return nil
}

func Sweep() {
	database.DB.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&models.Session{})
}
