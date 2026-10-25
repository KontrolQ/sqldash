package redishost

import (
	"context"
	"sqldash/models"
	"strconv"
)

type Statistics struct {
	StorageBytes int64
	MemoryBytes  int64
	Connections  int64
	Keys         int64
}

func StatisticsFor(ctx context.Context, record *models.Database, readKeys bool) (*Statistics, error) {
	instance, err := Instance(ctx, record)
	if err != nil {
		return nil, err
	}
	client, err := Client(ctx, record, Inspect)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	information, err := client.Info(ctx, "server", "clients", "memory").Result()
	if err != nil {
		return nil, Unavailable
	}
	fields := informationFields(information)
	if fields["run_id"] != instance.RunID {
		return nil, IdentityMismatch
	}
	memory, err := strconv.ParseInt(fields["used_memory"], 10, 64)
	if err != nil || memory < 0 {
		return nil, Unavailable
	}
	connections, err := strconv.ParseInt(fields["connected_clients"], 10, 64)
	if err != nil || connections < 0 {
		return nil, Unavailable
	}
	storage, err := StorageBytes(ctx, record)
	if err != nil {
		return nil, err
	}
	result := &Statistics{MemoryBytes: memory, Connections: connections, StorageBytes: storage, Keys: -1}
	if readKeys {
		result.Keys, err = client.DBSize(ctx).Result()
		if err != nil || result.Keys < 0 {
			return nil, Unavailable
		}
	}
	return result, nil
}
