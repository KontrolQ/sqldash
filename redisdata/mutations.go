package redisdata

import (
	"context"
	_ "embed"

	"github.com/redis/go-redis/v9"
)

//go:embed mutation.lua
var mutationScript string

func Mutate(ctx context.Context, client *redis.Client, mutation Mutation) (*Value, error) {
	return MutateBatch(ctx, client, []Mutation{mutation})
}

func Rename(ctx context.Context, client *redis.Client, key, target []byte, version string, replace bool) error {
	scope, err := requestScope(ctx)
	if err != nil {
		return err
	}
	if client == nil || string(key) == string(target) {
		return InvalidInput
	}
	if err := validKey(key); err != nil {
		return err
	}
	if err := validKey(target); err != nil {
		return err
	}
	expected, err := parseVersion(scope, key, version)
	if err != nil {
		return err
	}
	err = client.Watch(ctx, func(transaction *redis.Tx) error {
		current, err := currentSnapshot(ctx, transaction, key)
		if err != nil {
			return err
		}
		if !current.complete || current.kind == "none" || current.version() != expected {
			return StaleValue
		}
		if !replace {
			present, err := transaction.Exists(ctx, string(target)).Result()
			if err != nil {
				return commandError(err)
			}
			if present != 0 {
				return TargetExists
			}
		}
		_, err = transaction.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
			pipeline.Rename(ctx, string(key), string(target))
			return nil
		})
		return commandErrorOrNil(err)
	}, string(key), string(target))
	return commandErrorOrNil(err)
}

func Delete(ctx context.Context, client *redis.Client, references []KeyReference) error {
	scope, err := requestScope(ctx)
	if err != nil {
		return err
	}
	if client == nil || len(references) < 1 || len(references) > maximumReferences {
		return InvalidInput
	}
	keys := make([]string, 0, len(references))
	versions := make([]valueVersion, 0, len(references))
	seen := make(map[string]bool)
	for _, reference := range references {
		key, err := DecodeBytes(reference.Key)
		if err != nil {
			return err
		}
		if err := validKey(key); err != nil {
			return err
		}
		if seen[string(key)] {
			return InvalidInput
		}
		seen[string(key)] = true
		version, err := parseVersion(scope, key, reference.Version)
		if err != nil {
			return err
		}
		keys = append(keys, string(key))
		versions = append(versions, version)
	}
	err = client.Watch(ctx, func(transaction *redis.Tx) error {
		for index, key := range keys {
			current, err := currentSnapshot(ctx, transaction, []byte(key))
			if err != nil {
				return err
			}
			if !current.complete || current.kind == "none" || current.version() != versions[index] {
				return StaleValue
			}
		}
		_, err := transaction.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
			pipeline.Del(ctx, keys...)
			return nil
		})
		return commandErrorOrNil(err)
	}, keys...)
	return commandErrorOrNil(err)
}

func commandErrorOrNil(err error) error {
	if err == nil {
		return nil
	}
	return commandError(err)
}
