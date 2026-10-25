package redisdata

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Bytes struct {
	Encoding string  `json:"encoding"`
	Data     string  `json:"data"`
	Text     *string `json:"text,omitempty"`
}

type Scope struct {
	DatabaseID string
	Generation string
	SigningKey []byte
}

type ScanOptions struct {
	Cursor     string   `json:"cursor"`
	Match      string   `json:"match"`
	Type       string   `json:"type"`
	Limit      int      `json:"limit"`
	LoadedKeys []string `json:"loadedKeys"`
}

type KeySummary struct {
	Key  Bytes  `json:"key"`
	Type string `json:"type"`
}

type KeyPage struct {
	Keys         []KeySummary `json:"keys"`
	Cursor       string       `json:"cursor"`
	Complete     bool         `json:"complete"`
	Loaded       string       `json:"loaded"`
	DatabaseSize string       `json:"databaseSize"`
}

type ReadOptions struct {
	Cursor   string `json:"cursor"`
	Limit    int    `json:"limit"`
	MaxBytes int    `json:"maxBytes"`
}

type TTLState struct {
	Mode                  string `json:"mode"`
	Milliseconds          string `json:"milliseconds"`
	ExpiresAtMilliseconds string `json:"expiresAtMilliseconds"`
}

type HashEntry struct {
	Field Bytes `json:"field"`
	Value Bytes `json:"value"`
}

type ListEntry struct {
	Index string `json:"index"`
	Value Bytes  `json:"value"`
}

type ZSetEntry struct {
	Member Bytes  `json:"member"`
	Score  string `json:"score"`
}

type StreamEntry struct {
	ID     string      `json:"id"`
	Fields []HashEntry `json:"fields"`
}

type StreamGroup struct {
	Name            Bytes  `json:"name"`
	Consumers       string `json:"consumers"`
	Pending         string `json:"pending"`
	LastDeliveredID string `json:"lastDeliveredId"`
	EntriesRead     string `json:"entriesRead,omitempty"`
	Lag             string `json:"lag,omitempty"`
}

type JSONValue struct {
	Raw       Bytes  `json:"raw"`
	Formatted string `json:"formatted"`
}

type NativeProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type NativeValue struct {
	Summary    []NativeProperty `json:"summary"`
	Serialized *Bytes           `json:"serialized,omitempty"`
	Complete   bool             `json:"complete"`
}

type Value struct {
	Key            Bytes         `json:"key"`
	Type           string        `json:"type"`
	Exists         bool          `json:"exists"`
	TTL            TTLState      `json:"ttl"`
	Version        string        `json:"version"`
	Complete       bool          `json:"complete"`
	Truncated      bool          `json:"truncated"`
	Cursor         string        `json:"cursor"`
	Length         string        `json:"length"`
	Editable       bool          `json:"editable"`
	Representation string        `json:"representation"`
	RawFormat      string        `json:"rawFormat"`
	String         *Bytes        `json:"string,omitempty"`
	Hash           []HashEntry   `json:"hash,omitempty"`
	List           []ListEntry   `json:"list,omitempty"`
	Set            []Bytes       `json:"set,omitempty"`
	ZSet           []ZSetEntry   `json:"zset,omitempty"`
	Stream         []StreamEntry `json:"stream,omitempty"`
	Groups         []StreamGroup `json:"groups,omitempty"`
	JSON           *JSONValue    `json:"json,omitempty"`
	Native         *NativeValue  `json:"native,omitempty"`
}

type MutationKind string

const (
	Create        MutationKind = "create"
	SetString     MutationKind = "string_set"
	SetHash       MutationKind = "hash_set"
	DeleteHash    MutationKind = "hash_delete"
	SetList       MutationKind = "list_set"
	PushListLeft  MutationKind = "list_push_left"
	PushListRight MutationKind = "list_push_right"
	PopListLeft   MutationKind = "list_pop_left"
	PopListRight  MutationKind = "list_pop_right"
	AddSet        MutationKind = "set_add"
	RemoveSet     MutationKind = "set_remove"
	AddZSet       MutationKind = "zset_add"
	RemoveZSet    MutationKind = "zset_remove"
	AddStream     MutationKind = "stream_add"
	DeleteStream  MutationKind = "stream_delete"
	SetJSON       MutationKind = "json_set"
	ChangeTTL     MutationKind = "ttl"
	Copy          MutationKind = "copy"
)

type TTLChange struct {
	Mode                  string `json:"mode"`
	Milliseconds          string `json:"milliseconds,omitempty"`
	ExpiresAtMilliseconds string `json:"expiresAtMilliseconds,omitempty"`
}

type Mutation struct {
	Operation   MutationKind  `json:"operation"`
	Key         Bytes         `json:"key"`
	Version     string        `json:"version"`
	Type        string        `json:"type,omitempty"`
	Target      *Bytes        `json:"target,omitempty"`
	Replace     bool          `json:"replace"`
	Value       *Bytes        `json:"value,omitempty"`
	Hash        []HashEntry   `json:"hash,omitempty"`
	List        []Bytes       `json:"list,omitempty"`
	Members     []Bytes       `json:"members,omitempty"`
	SortedSet   []ZSetEntry   `json:"sortedSet,omitempty"`
	Stream      []StreamEntry `json:"stream,omitempty"`
	JSON        *Bytes        `json:"json,omitempty"`
	Index       string        `json:"index,omitempty"`
	Fields      []Bytes       `json:"fields,omitempty"`
	IDs         []string      `json:"ids,omitempty"`
	TTL         *TTLChange    `json:"ttl,omitempty"`
	ReadOptions ReadOptions   `json:"readOptions"`
}

type KeyReference struct {
	Key     Bytes  `json:"key"`
	Version string `json:"version"`
}

type DecodeOptions struct {
	MaxBytes int
	MaxRatio int
}

type scopeContextKey struct{}

func WithScope(ctx context.Context, scope Scope) context.Context {
	scope.SigningKey = append([]byte(nil), scope.SigningKey...)
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

var (
	_ func(context.Context, *redis.Client, ScanOptions) (*KeyPage, error)       = Scan
	_ func(context.Context, *redis.Client, []byte, ReadOptions) (*Value, error) = Read
	_ func(context.Context, *redis.Client, Mutation) (*Value, error)            = Mutate
	_ func(context.Context, *redis.Client, []byte, []byte, string, bool) error  = Rename
	_ func(context.Context, *redis.Client, []KeyReference) error                = Delete
)
