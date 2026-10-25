package redisdata

import (
	"strconv"
	"strings"
)

func (self snapshot) populate(value *Value) error {
	value.Representation = self.kind
	for _, raw := range self.data {
		switch self.kind {
		case "string":
			content, present := raw.(string)
			if !present {
				return Unsupported
			}
			encoded := EncodeBytes([]byte(content))
			value.String = &encoded
			value.Representation = "binary"
			if encoded.Text != nil {
				value.Representation = "text"
			}
			if self.complete {
				if formatted, err := FormatJSON([]byte(content)); err == nil {
					value.JSON = &JSONValue{Raw: encoded, Formatted: formatted}
					value.Representation = "json"
				}
			}
		case "hash", "list", "zset":
			pair, present := raw.([]any)
			if !present || len(pair) != 2 {
				return Unsupported
			}
			first, firstPresent := pair[0].(string)
			second, secondPresent := pair[1].(string)
			if !firstPresent || !secondPresent {
				return Unsupported
			}
			switch self.kind {
			case "hash":
				value.Hash = append(value.Hash, HashEntry{Field: EncodeBytes([]byte(first)), Value: EncodeBytes([]byte(second))})
			case "list":
				value.List = append(value.List, ListEntry{Index: first, Value: EncodeBytes([]byte(second))})
			case "zset":
				value.ZSet = append(value.ZSet, ZSetEntry{Member: EncodeBytes([]byte(first)), Score: second})
			}
		case "set":
			member, present := raw.(string)
			if !present {
				return Unsupported
			}
			value.Set = append(value.Set, EncodeBytes([]byte(member)))
		case "stream":
			entry, present := raw.([]any)
			if !present || len(entry) != 2 {
				return Unsupported
			}
			identity, present := entry[0].(string)
			if !present {
				return Unsupported
			}
			fields, present := entry[1].([]any)
			if !present || len(fields)%2 != 0 {
				return Unsupported
			}
			parsed := StreamEntry{ID: identity}
			for index := 0; index < len(fields); index += 2 {
				field, fieldPresent := fields[index].(string)
				content, contentPresent := fields[index+1].(string)
				if !fieldPresent || !contentPresent {
					return Unsupported
				}
				parsed.Fields = append(parsed.Fields, HashEntry{Field: EncodeBytes([]byte(field)), Value: EncodeBytes([]byte(content))})
			}
			value.Stream = append(value.Stream, parsed)
		case "ReJSON-RL":
			content, present := raw.(string)
			if !present {
				return Unsupported
			}
			formatted, err := FormatJSON([]byte(content))
			if err != nil {
				value.Representation = "json_raw"
			}
			value.JSON = &JSONValue{Raw: EncodeBytes([]byte(content)), Formatted: formatted}
		default:
			serialized, present := raw.(string)
			if !present {
				return Unsupported
			}
			if value.Native == nil {
				value.Native = &NativeValue{Complete: self.complete}
			}
			encoded := EncodeBytes([]byte(serialized))
			value.Native.Serialized = &encoded
		}
	}
	if self.kind == "stream" {
		value.Groups = self.groups
	} else if !coreType(self.kind) && self.kind != "none" {
		if value.Native == nil {
			value.Native = &NativeValue{Complete: self.complete}
		}
		for _, raw := range self.metadata {
			pair, present := raw.([]any)
			if !present || len(pair) != 2 {
				return Unsupported
			}
			name, namePresent := pair[0].(string)
			content, contentPresent := pair[1].(string)
			if !namePresent || !contentPresent {
				return Unsupported
			}
			value.Native.Summary = append(value.Native.Summary, NativeProperty{Name: name, Value: content})
		}
		value.Native.Summary = append(value.Native.Summary, self.nativeProperties...)
		value.Representation = "native"
	}
	return nil
}

func coreType(kind string) bool {
	switch kind {
	case "string", "hash", "list", "set", "zset", "stream", "ReJSON-RL":
		return true
	}
	return false
}

func validStreamID(identity string, automatic bool) bool {
	if automatic && identity == "*" {
		return true
	}
	parts := strings.Split(identity, "-")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		number, err := strconv.ParseUint(part, 10, 64)
		if err != nil || strconv.FormatUint(number, 10) != part {
			return false
		}
	}
	return identity != "0-0"
}
