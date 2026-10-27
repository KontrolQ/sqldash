package redis

import (
	"context"

	"sqldash/config"
)

func consoleContext(parent context.Context) (context.Context, context.CancelFunc) {
	if config.Redis.CommandTimeout > 0 {
		return context.WithTimeout(parent, config.Redis.CommandTimeout)
	}
	return context.WithCancel(parent)
}
