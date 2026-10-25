package redisdata

import (
	"bytes"
	"context"
	"github.com/redis/go-redis/v9"
)

func MutateBatch(ctx context.Context, client *redis.Client, mutations []Mutation) (*Value, error) {
	scope, err := requestScope(ctx)
	if err != nil {
		return nil, err
	}
	if client == nil || len(mutations) == 0 || len(mutations) > maximumReferences {
		return nil, InvalidInput
	}
	prepared := make([]preparedMutation, len(mutations))
	budget := maximumBytes * 2
	for index, mutation := range mutations {
		prepared[index], err = prepareMutation(mutation)
		if err != nil {
			return nil, err
		}
		if index > 0 && (!bytes.Equal(prepared[index].key, prepared[0].key) || mutation.Version != mutations[0].Version || mutation.Operation == Create || prepared[index].target != nil) {
			return nil, InvalidInput
		}
		if len(mutations) > 1 && (mutation.Operation == Create || prepared[index].target != nil) {
			return nil, InvalidInput
		}
		for _, argument := range prepared[index].command {
			switch value := argument.(type) {
			case string:
				budget -= len(value)
			case []byte:
				budget -= len(value)
			default:
				budget -= 32
			}
		}
		if budget < 0 {
			return nil, LimitExceeded
		}
	}
	first := prepared[0]
	var expected valueVersion
	if mutations[0].Operation != Create {
		expected, err = parseVersion(scope, first.key, mutations[0].Version)
		if err != nil {
			return nil, err
		}
	}
	watched := []string{string(first.key)}
	if first.target != nil {
		watched = append(watched, string(first.target))
	}
	err = client.Watch(ctx, func(transaction *redis.Tx) error {
		current, err := currentSnapshot(ctx, transaction, first.key)
		if err != nil {
			return err
		}
		if mutations[0].Operation == Create {
			if current.kind != "none" {
				return TargetExists
			}
		} else if !current.complete || current.version() != expected || current.kind == "none" {
			return StaleValue
		}
		for index, mutation := range mutations {
			if mutation.Operation != Create && prepared[index].kind != "" && prepared[index].kind != current.kind {
				return Unsupported
			}
		}
		if first.target != nil && !mutations[0].Replace {
			present, err := transaction.Exists(ctx, string(first.target)).Result()
			if err != nil {
				return commandError(err)
			}
			if present != 0 {
				return TargetExists
			}
		}
		_, err = transaction.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
			for _, mutation := range prepared {
				arguments := []any{"EVAL", mutationScript, len(watched)}
				for _, key := range watched {
					arguments = append(arguments, key)
				}
				arguments = append(arguments, mutation.ttlMode, mutation.ttlValue)
				arguments = append(arguments, mutation.command...)
				pipeline.Do(ctx, arguments...)
			}
			return nil
		})
		return commandErrorOrNil(err)
	}, watched...)
	if err != nil {
		return nil, commandError(err)
	}
	resultKey := first.key
	if first.target != nil {
		resultKey = first.target
	}
	return Read(ctx, client, resultKey, mutations[len(mutations)-1].ReadOptions)
}
