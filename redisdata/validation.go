package redisdata

import (
	"math"
	"strconv"
	"strings"
)

type preparedMutation struct {
	key      []byte
	target   []byte
	kind     string
	command  []any
	ttlMode  string
	ttlValue string
}

func prepareMutation(mutation Mutation) (preparedMutation, error) {
	prepared := preparedMutation{}
	var err error
	prepared.key, err = DecodeBytes(mutation.Key)
	if err != nil {
		return prepared, err
	}
	if err := validKey(prepared.key); err != nil {
		return prepared, err
	}
	prepared.ttlMode, prepared.ttlValue, err = ttlArguments(mutation.TTL)
	if err != nil {
		return prepared, err
	}
	if mutation.ReadOptions.Cursor != "" && mutation.ReadOptions.Cursor != "0" {
		return prepared, InvalidInput
	}
	if _, err := boundedOptions(mutation.ReadOptions); err != nil {
		return prepared, err
	}
	operation := mutation.Operation
	if operation == Create {
		if mutation.Version != "" || !coreType(mutation.Type) {
			return prepared, InvalidInput
		}
		prepared.kind = mutation.Type
		switch mutation.Type {
		case "string":
			operation = SetString
		case "hash":
			operation = SetHash
		case "list":
			operation = PushListRight
		case "set":
			operation = AddSet
		case "zset":
			operation = AddZSet
		case "stream":
			operation = AddStream
		case "ReJSON-RL":
			operation = SetJSON
		}
	} else if mutation.Type != "" {
		return prepared, InvalidInput
	}
	mask := 0
	if mutation.Value != nil {
		mask |= 1
	}
	if len(mutation.Hash) != 0 {
		mask |= 2
	}
	if len(mutation.List) != 0 {
		mask |= 4
	}
	if len(mutation.Members) != 0 {
		mask |= 8
	}
	if len(mutation.SortedSet) != 0 {
		mask |= 16
	}
	if len(mutation.Stream) != 0 {
		mask |= 32
	}
	if mutation.JSON != nil {
		mask |= 64
	}
	if mutation.Index != "" {
		mask |= 128
	}
	if len(mutation.Fields) != 0 {
		mask |= 256
	}
	if len(mutation.IDs) != 0 {
		mask |= 512
	}
	if mutation.Target != nil {
		mask |= 1024
	}
	if mutation.Replace {
		mask |= 2048
	}
	allowed := 0
	budget := 0
	decode := func(value Bytes) ([]byte, error) {
		decoded, err := DecodeBytes(value)
		if err != nil {
			return nil, err
		}
		budget += len(decoded)
		if budget > maximumBytes {
			return nil, LimitExceeded
		}
		return decoded, nil
	}
	members := func(values []Bytes) ([]any, error) {
		if len(values) < 1 || len(values) > maximumLimit {
			return nil, InvalidInput
		}
		arguments := make([]any, 0, len(values))
		for _, value := range values {
			decoded, err := decode(value)
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, decoded)
		}
		return arguments, nil
	}
	pairs := func(values []HashEntry) ([]any, error) {
		if len(values) < 1 || len(values) > maximumLimit {
			return nil, InvalidInput
		}
		arguments := make([]any, 0, len(values)*2)
		for _, value := range values {
			field, err := decode(value.Field)
			if err != nil {
				return nil, err
			}
			content, err := decode(value.Value)
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, field, content)
		}
		return arguments, nil
	}
	var arguments []any
	switch operation {
	case SetString:
		allowed = 1
		prepared.kind = "string"
		if mutation.Value == nil {
			return prepared, InvalidInput
		}
		value, err := decode(*mutation.Value)
		if err != nil {
			return prepared, err
		}
		prepared.command = []any{"SET", prepared.key, value, "KEEPTTL"}
	case SetHash:
		allowed = 2
		prepared.kind = "hash"
		arguments, err = pairs(mutation.Hash)
		prepared.command = []any{"HSET", prepared.key}
	case DeleteHash:
		allowed = 256
		prepared.kind = "hash"
		arguments, err = members(mutation.Fields)
		prepared.command = []any{"HDEL", prepared.key}
	case SetList:
		allowed = 1 | 128
		prepared.kind = "list"
		index, parseError := strconv.ParseInt(mutation.Index, 10, 64)
		if parseError != nil || strconv.FormatInt(index, 10) != mutation.Index || mutation.Value == nil {
			return prepared, InvalidInput
		}
		value, decodeError := decode(*mutation.Value)
		if decodeError != nil {
			return prepared, decodeError
		}
		prepared.command = []any{"LSET", prepared.key, mutation.Index, value}
	case PushListLeft, PushListRight:
		allowed = 4
		prepared.kind = "list"
		arguments, err = members(mutation.List)
		command := "RPUSH"
		if operation == PushListLeft {
			command = "LPUSH"
		}
		prepared.command = []any{command, prepared.key}
	case PopListLeft, PopListRight:
		prepared.kind = "list"
		command := "RPOP"
		if operation == PopListLeft {
			command = "LPOP"
		}
		prepared.command = []any{command, prepared.key}
	case AddSet, RemoveSet:
		allowed = 8
		prepared.kind = "set"
		arguments, err = members(mutation.Members)
		command := "SADD"
		if operation == RemoveSet {
			command = "SREM"
		}
		prepared.command = []any{command, prepared.key}
	case AddZSet:
		allowed = 16
		prepared.kind = "zset"
		if len(mutation.SortedSet) < 1 || len(mutation.SortedSet) > maximumLimit {
			return prepared, InvalidInput
		}
		prepared.command = []any{"ZADD", prepared.key}
		for _, entry := range mutation.SortedSet {
			if strings.TrimSpace(entry.Score) != entry.Score || len(entry.Score) > 128 || entry.Score == "" {
				return prepared, InvalidInput
			}
			score, parseError := strconv.ParseFloat(entry.Score, 64)
			if parseError != nil || math.IsNaN(score) || (math.IsInf(score, 0) && entry.Score != "inf" && entry.Score != "+inf" && entry.Score != "-inf") {
				return prepared, InvalidInput
			}
			member, decodeError := decode(entry.Member)
			if decodeError != nil {
				return prepared, decodeError
			}
			prepared.command = append(prepared.command, entry.Score, member)
		}
	case RemoveZSet:
		allowed = 8
		prepared.kind = "zset"
		arguments, err = members(mutation.Members)
		prepared.command = []any{"ZREM", prepared.key}
	case AddStream:
		allowed = 32
		prepared.kind = "stream"
		if len(mutation.Stream) != 1 || !validStreamID(mutation.Stream[0].ID, true) {
			return prepared, InvalidInput
		}
		arguments, err = pairs(mutation.Stream[0].Fields)
		prepared.command = []any{"XADD", prepared.key, mutation.Stream[0].ID}
	case DeleteStream:
		allowed = 512
		prepared.kind = "stream"
		if len(mutation.IDs) < 1 || len(mutation.IDs) > maximumLimit {
			return prepared, InvalidInput
		}
		prepared.command = []any{"XDEL", prepared.key}
		for _, identity := range mutation.IDs {
			if !validStreamID(identity, false) {
				return prepared, InvalidInput
			}
			prepared.command = append(prepared.command, identity)
		}
	case SetJSON:
		allowed = 64
		prepared.kind = "ReJSON-RL"
		if mutation.JSON == nil {
			return prepared, InvalidInput
		}
		value, decodeError := decode(*mutation.JSON)
		if decodeError != nil {
			return prepared, decodeError
		}
		if _, formatError := FormatJSON(value); formatError != nil {
			return prepared, formatError
		}
		prepared.command = []any{"JSON.SET", prepared.key, "$", value}
	case ChangeTTL:
		if mutation.TTL == nil || mutation.TTL.Mode == "preserve" {
			return prepared, InvalidInput
		}
	case Copy:
		allowed = 1024 | 2048
		if mutation.Target == nil {
			return prepared, InvalidInput
		}
		prepared.target, err = DecodeBytes(*mutation.Target)
		if err != nil {
			return prepared, err
		}
		if err := validKey(prepared.target); err != nil {
			return prepared, err
		}
		if string(prepared.key) == string(prepared.target) {
			return prepared, InvalidInput
		}
		prepared.command = []any{"COPY", prepared.key, prepared.target}
		if mutation.Replace {
			prepared.command = append(prepared.command, "REPLACE")
		}
	default:
		return prepared, Unsupported
	}
	if err != nil {
		return prepared, err
	}
	if mask & ^allowed != 0 {
		return prepared, InvalidInput
	}
	prepared.command = append(prepared.command, arguments...)
	return prepared, nil
}
