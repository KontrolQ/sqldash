package redisconnections

import (
	"context"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
)

func upgradePrimaryPubSub(ctx context.Context, record *models.Database, connection *models.RedisConnection, instance *models.RedisInstance) error {
	if !connection.PrimaryConnection || connection.State != models.DatabaseReady {
		return nil
	}
	issuer, err := access.LoadWith(database.DB.WithContext(ctx), connection.AccountID)
	if err != nil {
		return Unavailable
	}
	subscribe := connection.Subscribe || issuer.Can(permissions.RedisSubscribe)
	publish := connection.Publish || issuer.Can(permissions.RedisPublish)
	if subscribe == connection.Subscribe && publish == connection.Publish {
		return nil
	}
	if connection.PendingSecret != "" {
		if _, err := recoverPrimary(connection, instance); err != nil {
			return err
		}
		if err := clearPending(ctx, connection, connection.State); err != nil {
			return err
		}
	}
	updated := database.DB.WithContext(ctx).Model(&models.RedisConnection{}).Where("id = ? AND database_id = ? AND instance_identity = ? AND generation = ? AND state = ? AND subscribe = ? AND publish = ? AND revoked_at IS NULL", connection.ID, record.ID, instance.Identity, connection.Generation, connection.State, connection.Subscribe, connection.Publish).Updates(map[string]any{"subscribe": subscribe, "publish": publish})
	if updated.Error != nil || updated.RowsAffected != 1 {
		return Conflict
	}
	connection.Subscribe, connection.Publish = subscribe, publish
	return nil
}
