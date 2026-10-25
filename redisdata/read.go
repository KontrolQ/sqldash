package redisdata

import (
	"context"
	_ "embed"
	"errors"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

//go:embed snapshot.lua
var snapshotScript string

type snapshot struct {
	kind             string
	length           string
	position         string
	offset           int
	complete         bool
	data             []any
	metadata         []any
	digest           string
	expiry           string
	ttl              string
	groups           []StreamGroup
	nativeProperties []NativeProperty
}

func Read(ctx context.Context, client *redis.Client, key []byte, options ReadOptions) (*Value, error) {
	scope, err := requestScope(ctx)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, InvalidInput
	}
	if err := validKey(key); err != nil {
		return nil, err
	}
	options, err = boundedOptions(options)
	if err != nil {
		return nil, err
	}
	cursor, err := cursorFor(scope, key, options.Cursor)
	if err != nil {
		return nil, err
	}
	var loaded snapshot
	for attempt := 0; attempt < 3; attempt++ {
		err = client.Watch(ctx, func(transaction *redis.Tx) error {
			var err error
			loaded, err = loadSnapshot(ctx, transaction, key, cursor, options)
			if err != nil {
				return err
			}
			_, err = transaction.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
				pipeline.Exists(ctx, string(key))
				return nil
			})
			return err
		}, string(key))
		if !errors.Is(err, redis.TxFailedErr) {
			break
		}
	}
	if err != nil {
		return nil, commandError(err)
	}
	if cursor.Type != "" && loaded.kind != cursor.Type {
		return nil, StaleValue
	}
	return loaded.value(scope, key)
}

func queueSnapshot(ctx context.Context, pipeline redis.Pipeliner, key []byte, cursor readCursor, options ReadOptions) (*redis.Cmd, *redis.Cmd, *redis.Cmd) {
	contents := pipeline.Do(ctx, "EVAL_RO", snapshotScript, 1, key, cursor.Position, cursor.Offset, options.Limit, options.MaxBytes)
	expiry := pipeline.Do(ctx, "PEXPIRETIME", key)
	ttl := pipeline.Do(ctx, "PTTL", key)
	return contents, expiry, ttl
}

func currentSnapshot(ctx context.Context, transaction *redis.Tx, key []byte) (snapshot, error) {
	return loadSnapshot(ctx, transaction, key, readCursor{Position: "0"}, ReadOptions{Limit: maximumLimit, MaxBytes: maximumBytes})
}

func loadSnapshot(ctx context.Context, transaction *redis.Tx, key []byte, cursor readCursor, options ReadOptions) (snapshot, error) {
	pipeline := transaction.Pipeline()
	contents, expiry, ttl := queueSnapshot(ctx, pipeline, key, cursor, options)
	if _, err := pipeline.Exec(ctx); err != nil {
		return snapshot{}, commandError(err)
	}
	loaded, err := parseSnapshot(contents, expiry, ttl)
	if err != nil {
		return snapshot{}, err
	}
	if loaded.kind == "stream" && loaded.complete {
		if err := loaded.loadStreamMetadata(ctx, transaction, key, options); err != nil {
			return snapshot{}, err
		}
	} else if !coreType(loaded.kind) && loaded.kind != "none" {
		if err := loaded.loadNativeMetadata(ctx, transaction, key, options); err != nil {
			return snapshot{}, err
		}
	}
	return loaded, nil
}

func parseSnapshot(contents, expiry, ttl *redis.Cmd) (snapshot, error) {
	values, err := contents.Slice()
	if err != nil || len(values) != 8 {
		return snapshot{}, commandError(err)
	}
	kind, first := values[0].(string)
	length, second := values[1].(string)
	position, third := values[2].(string)
	offset, fourth := values[3].(string)
	complete, fifth := values[4].(string)
	data, sixth := values[5].([]any)
	metadata, seventh := values[6].([]any)
	digest, eighth := values[7].(string)
	if !first || !second || !third || !fourth || !fifth || !sixth || !seventh || !eighth {
		return snapshot{}, Unsupported
	}
	offsetNumber, err := strconv.Atoi(offset)
	if err != nil {
		return snapshot{}, Unsupported
	}
	expiryNumber, err := expiry.Int64()
	if err != nil {
		return snapshot{}, commandError(err)
	}
	ttlNumber, err := ttl.Int64()
	if err != nil {
		return snapshot{}, commandError(err)
	}
	return snapshot{kind: kind, length: length, position: position, offset: offsetNumber, complete: complete == "1", data: data, metadata: metadata, digest: digest, expiry: strconv.FormatInt(expiryNumber, 10), ttl: strconv.FormatInt(ttlNumber, 10)}, nil
}

func (self snapshot) version() valueVersion {
	return valueVersion{Type: self.kind, Digest: self.digest, Expiry: self.expiry}
}

func (self snapshot) value(scope Scope, key []byte) (*Value, error) {
	value := &Value{Key: EncodeBytes(key), Type: self.kind, Exists: self.kind != "none", Complete: self.complete, Truncated: !self.complete, Length: self.length, RawFormat: "sqldash.redis.value.v1", TTL: TTLState{Mode: "expiring", Milliseconds: self.ttl, ExpiresAtMilliseconds: self.expiry}}
	if self.kind == "none" {
		value.TTL.Mode = "missing"
		return value, nil
	}
	if self.expiry == "-1" {
		value.TTL.Mode = "persistent"
	}
	if self.position != "" {
		value.Cursor = seal(scope, "cursor", key, readCursor{Type: self.kind, Position: self.position, Offset: self.offset})
	}
	if self.complete && self.digest != "" {
		value.Version = seal(scope, "version", key, self.version())
	}
	if err := self.populate(value); err != nil {
		return nil, err
	}
	value.Editable = self.complete && value.Version != "" && value.Native == nil && value.Representation != "json_raw"
	return value, nil
}

func commandError(err error) error {
	if err == nil {
		return Unsupported
	}
	if errors.Is(err, redis.TxFailedErr) || strings.Contains(err.Error(), "SQLDASH_STALE") {
		return StaleValue
	}
	if strings.Contains(err.Error(), "SQLDASH_LIMIT") {
		return LimitExceeded
	}
	if strings.Contains(err.Error(), "SQLDASH_INPUT") {
		return InvalidInput
	}
	if strings.Contains(err.Error(), "SQLDASH_TARGET") {
		return TargetExists
	}
	if strings.HasPrefix(err.Error(), "WRONGTYPE") {
		return StaleValue
	}
	return err
}
