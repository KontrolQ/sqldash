package redis

import (
	"context"

	"sqldash/config"
	"sqldash/redisdata"

	"github.com/gofiber/fiber/v3"
)

type ScanRequest struct {
	redisdata.ScanOptions
	MatchBytes *redisdata.Bytes `json:"matchBytes,omitempty"`
}

type ReadRequest struct {
	Key     redisdata.Bytes       `json:"key"`
	Options redisdata.ReadOptions `json:"options"`
}

type BatchRequest struct {
	Mutations []redisdata.Mutation `json:"mutations"`
}

func MutateBatch(ctx context.Context, name string, asked BatchRequest) (*redisdata.Value, *fiber.Error) {
	if len(asked.Mutations) == 0 || len(asked.Mutations) > 100 {
		return nil, fiber.NewError(400, InvalidRequest)
	}
	fresh, client, err := dataClient(ctx, name, true)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	for index := range asked.Mutations {
		asked.Mutations[index].ReadOptions = boundedRead(asked.Mutations[index].ReadOptions)
	}
	value, mutationError := redisdata.MutateBatch(fresh, client, asked.Mutations)
	return value, operationError(mutationError)
}

type RenameRequest struct {
	Key     redisdata.Bytes `json:"key"`
	Target  redisdata.Bytes `json:"target"`
	Version string          `json:"version"`
	Replace bool            `json:"replace"`
}

type DeleteRequest struct {
	Keys []redisdata.KeyReference `json:"keys"`
}

func Scan(ctx context.Context, name string, asked ScanRequest) (*redisdata.KeyPage, *fiber.Error) {
	options := asked.ScanOptions
	if asked.MatchBytes != nil {
		decoded, err := redisdata.DecodeBytes(*asked.MatchBytes)
		if err != nil {
			return nil, operationError(err)
		}
		options.Match = string(decoded)
	}
	fresh, client, err := dataClient(ctx, name, false)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	if options.Limit <= 0 || options.Limit > 500 {
		options.Limit = 200
	}
	if int64(options.Limit) > config.Redis.MaximumScanBatch && config.Redis.MaximumScanBatch > 0 {
		options.Limit = int(config.Redis.MaximumScanBatch)
	}
	result, scanError := redisdata.Scan(fresh, client, options)
	return result, operationError(scanError)
}

func Read(ctx context.Context, name string, asked ReadRequest) (*redisdata.Value, *fiber.Error) {
	key, decodeError := redisdata.DecodeBytes(asked.Key)
	if decodeError != nil {
		return nil, operationError(decodeError)
	}
	fresh, client, err := dataClient(ctx, name, false)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	asked.Options = boundedRead(asked.Options)
	result, readError := redisdata.Read(fresh, client, key, asked.Options)
	return result, operationError(readError)
}

func Mutate(ctx context.Context, name string, asked redisdata.Mutation) (*redisdata.Value, *fiber.Error) {
	fresh, client, err := dataClient(ctx, name, true)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	asked.ReadOptions = boundedRead(asked.ReadOptions)
	result, mutationError := redisdata.Mutate(fresh, client, asked)
	return result, operationError(mutationError)
}

func Rename(ctx context.Context, name string, asked RenameRequest) *fiber.Error {
	key, decodeError := redisdata.DecodeBytes(asked.Key)
	if decodeError != nil {
		return operationError(decodeError)
	}
	target, decodeError := redisdata.DecodeBytes(asked.Target)
	if decodeError != nil {
		return operationError(decodeError)
	}
	fresh, client, err := dataClient(ctx, name, true)
	if err != nil {
		return err
	}
	defer client.Close()
	return operationError(redisdata.Rename(fresh, client, key, target, asked.Version, asked.Replace))
}

func Delete(ctx context.Context, name string, asked DeleteRequest) *fiber.Error {
	if len(asked.Keys) < 1 || len(asked.Keys) > 100 {
		return fiber.NewError(400, InvalidRequest)
	}
	fresh, client, err := dataClient(ctx, name, true)
	if err != nil {
		return err
	}
	defer client.Close()
	return operationError(redisdata.Delete(fresh, client, asked.Keys))
}

func boundedRead(options redisdata.ReadOptions) redisdata.ReadOptions {
	if options.Limit <= 0 || options.Limit > 500 {
		options.Limit = 200
	}
	limit := 256 * 1024
	if config.Redis.MaximumValueBytes > 0 && config.Redis.MaximumValueBytes < int64(limit) {
		limit = int(config.Redis.MaximumValueBytes)
	}
	if options.MaxBytes <= 0 || options.MaxBytes > limit {
		options.MaxBytes = limit
	}
	return options
}
