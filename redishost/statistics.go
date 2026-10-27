package redishost

import (
	"context"
	"sqldash/models"
	"strconv"
)

type Statistics struct {
	StorageBytes      int64
	MemoryBytes       int64
	Connections       int64
	Keys              int64
	CommandsProcessed int64
	KeyspaceHits      int64
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
	information, err := client.Info(ctx, "server", "clients", "memory", "stats").Result()
	if err != nil {
		return nil, Unavailable
	}
	fields := informationFields(information)
	if fields["run_id"] != instance.RunID {
		return nil, IdentityMismatch
	}
	result, err := parseStatistics(fields)
	if err != nil {
		return nil, err
	}
	result.StorageBytes, err = StorageBytes(ctx, record)
	if err != nil {
		return nil, err
	}
	if readKeys {
		result.Keys, err = client.DBSize(ctx).Result()
		if err != nil || result.Keys < 0 {
			return nil, Unavailable
		}
	}
	return result, nil
}

func parseStatistics(fields map[string]string) (*Statistics, error) {
	result := &Statistics{Keys: -1}
	for field, destination := range map[string]*int64{
		"used_memory":              &result.MemoryBytes,
		"connected_clients":        &result.Connections,
		"total_commands_processed": &result.CommandsProcessed,
		"keyspace_hits":            &result.KeyspaceHits,
	} {
		value, err := strconv.ParseInt(fields[field], 10, 64)
		if err != nil || value < 0 {
			return nil, Unavailable
		}
		*destination = value
	}
	return result, nil
}
