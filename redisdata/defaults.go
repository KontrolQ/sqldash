package redisdata

import "time"

const (
	defaultLimit       = 100
	maximumLimit       = 1000
	defaultBytes       = 64 << 10
	maximumBytes       = 256 << 10
	maximumKeyBytes    = 8192
	maximumLoadedKeys  = 10000
	maximumReferences  = 100
	maximumTokenBytes  = 16384
	maximumDecodeRatio = 100
	maximumDecodeTime  = 250 * time.Millisecond
)
