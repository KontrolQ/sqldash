package redisdata

import (
	"context"
	_ "embed"
	"strconv"

	"github.com/redis/go-redis/v9"
)

//go:embed scan.lua
var scanScript string

func Scan(ctx context.Context, client *redis.Client, options ScanOptions) (*KeyPage, error) {
	if _, err := requestScope(ctx); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, InvalidInput
	}
	if options.Cursor == "" {
		options.Cursor = "0"
	}
	if _, err := parseUnsigned(options.Cursor); err != nil {
		return nil, err
	}
	if options.Limit == 0 {
		options.Limit = defaultLimit
	}
	if options.Limit < 1 || options.Limit > maximumLimit || len(options.Match) > maximumKeyBytes || len(options.Type) > 128 || len(options.LoadedKeys) > maximumLoadedKeys {
		return nil, LimitExceeded
	}
	loaded := make(map[string]bool, len(options.LoadedKeys))
	loadedBytes := 0
	for _, encoded := range options.LoadedKeys {
		key, err := DecodeBytes(Bytes{Encoding: "base64", Data: encoded})
		if err != nil {
			return nil, err
		}
		if err := validKey(key); err != nil {
			return nil, err
		}
		loadedBytes += len(key)
		if loadedBytes > maximumBytes*8 {
			return nil, LimitExceeded
		}
		loaded[string(key)] = true
	}
	pipeline := client.Pipeline()
	command := pipeline.Do(ctx, "EVAL_RO", scanScript, 0, options.Cursor, options.Limit, options.Match, options.Type)
	size := pipeline.DBSize(ctx)
	if _, err := pipeline.Exec(ctx); err != nil {
		return nil, commandError(err)
	}
	response, err := command.Slice()
	if err != nil || len(response) != 2 {
		return nil, Unsupported
	}
	cursor, cursorPresent := response[0].(string)
	entries, entriesPresent := response[1].([]any)
	if !cursorPresent || !entriesPresent {
		return nil, Unsupported
	}
	page := &KeyPage{Keys: []KeySummary{}, Cursor: cursor, Complete: cursor == "0", DatabaseSize: strconv.FormatInt(size.Val(), 10)}
	for _, raw := range entries {
		entry, present := raw.([]any)
		if !present || len(entry) != 2 {
			return nil, Unsupported
		}
		key, keyPresent := entry[0].(string)
		kind, kindPresent := entry[1].(string)
		if !keyPresent || !kindPresent {
			return nil, Unsupported
		}
		if !loaded[key] {
			loaded[key] = true
			if len(loaded) > maximumLoadedKeys {
				return nil, LimitExceeded
			}
			page.Keys = append(page.Keys, KeySummary{Key: EncodeBytes([]byte(key)), Type: kind})
		}
	}
	page.Loaded = strconv.Itoa(len(loaded))
	return page, nil
}
