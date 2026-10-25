package redisconnections

import (
	"context"
	"strconv"

	"sqldash/database"
	"sqldash/models"
	"sqldash/redishost"
)

func retirePrimary(ctx context.Context, record *models.Database, connection *models.RedisConnection) error {
	username := connection.Username
	if username == record.Name {
		username = "sqd_" + strconv.FormatUint(uint64(record.ID), 10) + "_retired_" + strconv.FormatUint(uint64(connection.ID), 10)
	}
	updated := database.DB.WithContext(ctx).Unscoped().Model(&models.RedisConnection{}).Where("id = ? AND generation = ? AND instance_identity = ? AND username = ? AND primary_connection = ?", connection.ID, connection.Generation, connection.InstanceIdentity, connection.Username, true).Updates(map[string]any{"primary_connection": false, "username": username})
	if updated.Error != nil || updated.RowsAffected != 1 {
		return Conflict
	}
	connection.PrimaryConnection = false
	connection.Username = username
	return nil
}

func renamePrimary(ctx context.Context, record *models.Database, connection *models.RedisConnection, instance *models.RedisInstance) error {
	if !connection.PrimaryConnection || connection.DeletedAt.Valid || connection.Username == record.Name {
		return nil
	}
	// Older databases could use this name while their public ACL user was generated independently.
	if record.Name == "default" {
		return nil
	}
	if !models.PublicRedisUsername(record.Name) {
		return Conflict
	}
	var count int64
	if err := database.DB.WithContext(ctx).Unscoped().Model(&models.RedisConnection{}).Where("username = ? AND id <> ?", record.Name, connection.ID).Count(&count).Error; err != nil {
		return Unavailable
	}
	if count != 0 {
		return Conflict
	}
	// Retire the durable ACL and its sessions before forgetting the old name, so retries cannot orphan it.
	if err := redishost.RemoveUser(ctx, record, connection.Username); err != nil {
		return quarantine(ctx, record, instance)
	}
	updated := database.DB.WithContext(ctx).Model(&models.RedisConnection{}).Where("id = ? AND database_id = ? AND instance_identity = ? AND generation = ? AND username = ? AND primary_connection = ? AND state = ?", connection.ID, record.ID, instance.Identity, connection.Generation, connection.Username, true, connection.State).Update("username", record.Name)
	if updated.Error != nil || updated.RowsAffected != 1 {
		return Conflict
	}
	connection.Username = record.Name
	return nil
}
