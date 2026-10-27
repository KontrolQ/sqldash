package pgmetrics

import "time"

const (
	SampleVersion  = 1
	SampleInterval = time.Minute
	SampleTimeout  = 20 * time.Second
	LogPrefix      = "PostgreSQL metrics"
)
