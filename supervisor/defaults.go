package supervisor

import "time"

const (
	LogPrefix = "Supervisor"

	RestartDelay        = time.Second
	MaximumRestartDelay = 30 * time.Second
	HealthyRunDuration  = 10 * time.Second
	ShutdownGrace       = 10 * time.Second
)
