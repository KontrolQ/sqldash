package rediswire

type Value struct {
	Kind   byte
	Bytes  []byte
	Values []Value
	Null   bool
}

type Limits struct {
	MaximumBytes     int64
	MaximumBulkBytes int64
	MaximumElements  int
	MaximumDepth     int
	MaximumLineBytes int
}

type RelayOptions struct {
	MaximumBulkBytes   int64
	MaximumArguments   int
	MaximumHeaderBytes int
	Inspect            func([][]byte) error
}
