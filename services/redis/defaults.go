package redis

import "time"

const (
	maximumRequestBytes   = 2 * 1024 * 1024
	maximumReplyBytes     = 2 * 1024 * 1024
	maximumReplyElements  = 10000
	maximumLiveBytes      = 64 * 1024
	maximumHistorySamples = 360
	historyInterval       = 10 * time.Second
	streamLifetime        = 30 * time.Minute
)

func RequestLimit() int { return maximumRequestBytes }
