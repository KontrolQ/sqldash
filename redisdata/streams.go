package redisdata

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func (self *snapshot) loadStreamMetadata(ctx context.Context, transaction *redis.Tx, key []byte, options ReadOptions) error {
	pipeline := transaction.Pipeline()
	informationCommand := pipeline.Do(ctx, "XINFO", "STREAM", key)
	groupsCommand := pipeline.Do(ctx, "XINFO", "GROUPS", key)
	if _, err := pipeline.Exec(ctx); err != nil {
		return commandError(err)
	}
	information, err := informationCommand.Slice()
	if err != nil {
		return commandError(err)
	}
	groups, err := groupsCommand.Slice()
	if err != nil {
		return commandError(err)
	}
	if len(groups) > options.Limit {
		return LimitExceeded
	}
	budget := options.MaxBytes
	if !boundedReply(information, &budget) || !boundedReply(groups, &budget) {
		return LimitExceeded
	}
	for _, raw := range groups {
		properties, present := raw.([]any)
		if !present || len(properties)%2 != 0 {
			return Unsupported
		}
		group := StreamGroup{}
		for index := 0; index < len(properties); index += 2 {
			name, present := properties[index].(string)
			if !present {
				return Unsupported
			}
			value := properties[index+1]
			text := ""
			switch value := value.(type) {
			case string:
				text = value
			case int64:
				text = strconv.FormatInt(value, 10)
			case nil:
			default:
				return Unsupported
			}
			switch name {
			case "name":
				group.Name = EncodeBytes([]byte(text))
			case "consumers":
				group.Consumers = text
			case "pending":
				group.Pending = text
			case "last-delivered-id":
				group.LastDeliveredID = text
			case "entries-read":
				group.EntriesRead = text
			case "lag":
				group.Lag = text
			}
		}
		self.groups = append(self.groups, group)
	}
	digest := sha1.New()
	fingerprint(digest, self.digest)
	fingerprint(digest, information)
	fingerprint(digest, groups)
	self.digest = hex.EncodeToString(digest.Sum(nil))
	return nil
}

func boundedReply(value any, budget *int) bool {
	*budget -= 1
	if *budget < 0 {
		return false
	}
	switch value := value.(type) {
	case string:
		*budget -= len(value)
	case []any:
		for _, child := range value {
			if !boundedReply(child, budget) {
				return false
			}
		}
	case int64:
		*budget -= 20
	case nil:
	default:
		return false
	}
	return *budget >= 0
}

func fingerprint(digest hash.Hash, value any) {
	switch value := value.(type) {
	case string:
		digest.Write([]byte("s" + strconv.Itoa(len(value)) + ":"))
		digest.Write([]byte(value))
	case int64:
		digest.Write([]byte("i" + strconv.FormatInt(value, 10) + ":"))
	case []any:
		digest.Write([]byte("a" + strconv.Itoa(len(value)) + ":"))
		for _, child := range value {
			fingerprint(digest, child)
		}
	case nil:
		digest.Write([]byte("n"))
	}
}
