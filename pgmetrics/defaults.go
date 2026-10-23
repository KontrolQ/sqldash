package pgmetrics

import "time"

const (
	SampleVersion  = 1
	SampleInterval = time.Minute
	SampleTimeout  = 20 * time.Second
	Retention      = 91 * 24 * time.Hour
	LogPrefix      = "PostgreSQL metrics"
)
