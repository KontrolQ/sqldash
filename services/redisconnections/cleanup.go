package redisconnections

import (
	"context"

	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func Clear(ctx context.Context, databaseID uint) error {
	if databaseID == 0 {
		return Invalid
	}
	return fenced(ctx, databaseID, func() error {
		var connections []models.RedisConnection
		return database.DB.WithContext(ctx).Unscoped().Where("database_id = ?", databaseID).Order("id asc").FindInBatches(&connections, 100, func(_ *gorm.DB, _ int) error {
			for index := range connections {
				connection := &connections[index]
				if err := clearPrimary(connection); err != nil {
					return err
				}
				if err := clearPending(ctx, connection, connection.State); err != nil {
					return err
				}
			}
			return nil
		}).Error
	})
}
