package redis

import (
	"context"
	"time"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/redishost"
)

type Backend struct{}

var _ engines.Backend = (*Backend)(nil)

func init() { engines.Register(models.EngineRedis, &Backend{}) }

func (self *Backend) Features() engines.Features {
	return engines.Features{TypedValues: true, NativeConnections: true, Activity: true}
}

func (self *Backend) Create(ctx context.Context, record *models.Database) error {
	return redishost.Create(ctx, record)
}

func (self *Backend) Delete(ctx context.Context, record *models.Database) error {
	return redishost.Delete(ctx, record)
}

func (self *Backend) Run(context.Context, *models.Database, engines.Mode, ...engines.Statement) ([]*engines.Result, error) {
	return nil, engines.Unsupported
}

func (self *Backend) Statistics(ctx context.Context, record *models.Database) (*engines.StatisticsRecord, error) {
	size, err := redishost.StorageBytes(ctx, record)
	if err != nil {
		return nil, err
	}
	return &engines.StatisticsRecord{Identifier: record.ClusterID, StorageBytesUsed: size}, nil
}

func (self *Backend) Configuration(context.Context, *models.Database) (*engines.ConfigurationRecord, error) {
	return nil, engines.Unsupported
}

func (self *Backend) SetConfiguration(context.Context, *models.Database, engines.ConfigurationRecord) error {
	return engines.Unsupported
}

func (self *Backend) Fork(context.Context, *models.Database, *models.Database, *time.Time) error {
	return engines.Unsupported
}
