package redis

import "time"

const (
	maximumReplyBytes     = 2 * 1024 * 1024
	maximumLiveBytes      = 64 * 1024
	maximumHistorySamples = 360
	historyInterval       = 10 * time.Second
)
