package transferpolicy

import (
	"context"
	"time"

	"sqldash/config"
)

func ByteLimit(requested int64) int64 {
	return stricter(requested, config.Transfer.MaximumBytes)
}

func Context(parent context.Context, requested time.Duration) (context.Context, context.CancelFunc) {
	timeout := time.Duration(stricter(int64(requested), int64(config.Transfer.Timeout)))
	if timeout > 0 {
		return context.WithTimeout(parent, timeout)
	}
	return context.WithCancel(parent)
}

func stricter(requested, configured int64) int64 {
	if requested == 0 {
		return configured
	}
	if configured == 0 {
		return requested
	}
	return min(requested, configured)
}
