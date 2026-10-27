package databases

import (
	"context"
	"errors"

	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func completeOperation(ctx context.Context, record *models.Database, operation *models.DatabaseOperation) error {
	return database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if _, _, err := operationAuthorityWith(ctx, transaction, record, operation); err != nil {
			return err
		}
		if record.Kind() == models.EngineRedis {
			var credentials int64
			if err := transaction.Model(&models.RedisConnection{}).Where("database_id = ? AND instance_identity = ? AND primary_connection = ? AND state = ? AND scope = ? AND revoked_at IS NULL", record.ID, record.ClusterID, true, models.DatabaseReady, models.ScopeReadWrite).Count(&credentials).Error; err != nil || credentials != 1 {
				return errors.New(DatabaseNotReady)
			}
			published := transaction.Model(&models.RedisInstance{}).Where("database_id = ? AND identity = ? AND state = ?", record.ID, record.ClusterID, models.DatabaseReady).Update("published", true)
			if published.Error != nil {
				return published.Error
			}
			if published.RowsAffected != 1 {
				return errors.New(DatabaseNotReady)
			}
		}
		completion := transaction.Model(operation).Where("state = ? AND lease_until = ?", models.DatabaseApplying, operation.LeaseUntil).Updates(map[string]any{"state": models.DatabaseReady, "lease_until": nil, "heartbeat_until": nil, "failure": ""})
		if completion.Error != nil {
			return completion.Error
		}
		if completion.RowsAffected != 1 {
			return errors.New(DatabaseNotReady)
		}
		result := transaction.Model(record).Where("state <> ?", models.DatabaseDeleting).Updates(map[string]any{"state": models.DatabaseReady, "backend_oid": record.BackendOID, "cluster_id": record.ClusterID, "policy_version": gorm.Expr("policy_version + 1")})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New(DatabaseNotReady)
		}
		return nil
	})
}
