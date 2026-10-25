package redisaccess

import (
	"context"
	"errors"

	"sqldash/database"
	"sqldash/models"
	"sqldash/redisgateway"
	"sqldash/redishost"

	"gorm.io/gorm"
)

func Terminate(ctx context.Context, binding *redisgateway.Binding) error {
	if binding == nil || binding.ClientID < 1 || !models.PublicRedisUsername(binding.Username) {
		return Refused
	}
	snapshot, valid := binding.State.(*authority)
	if !valid || snapshot == nil {
		return Refused
	}
	var record models.Database
	if err := database.DB.WithContext(ctx).First(&record, binding.DatabaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return Refused
	}
	if record.Kind() != models.EngineRedis {
		return Refused
	}
	var instance models.RedisInstance
	if err := database.DB.WithContext(ctx).Where("database_id = ?", record.ID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return Refused
	}
	if instance.Identity != snapshot.InstanceIdentity || instance.Generation != snapshot.InstanceGeneration || instance.RunID != snapshot.RunID {
		return nil
	}
	return redishost.KillClientForRun(ctx, &record, binding.ClientID, snapshot.RunID)
}

func Unsafe(ctx context.Context, binding *redisgateway.Binding, failure error) {
	if binding == nil {
		panic(Refused)
	}
	snapshot, valid := binding.State.(*authority)
	if !valid || snapshot == nil {
		panic(Refused)
	}
	record := &models.Database{Model: gorm.Model{ID: binding.DatabaseID}, Engine: models.EngineRedis, ClusterID: snapshot.InstanceIdentity}
	if err := redishost.Quarantine(ctx, record, snapshot.InstanceGeneration); err != nil {
		panic(err)
	}
}
