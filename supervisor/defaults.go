package supervisor

import "time"

const (
	LogPrefix = "Supervisor"

	RestartDelay          = time.Second
	MaximumRestartDelay   = 30 * time.Second
	HealthyRunDuration    = 10 * time.Second
	ShutdownGrace         = 10 * time.Second
	ReadinessTimeout      = 60 * time.Second
	ReadinessInterval     = 250 * time.Millisecond
	StopConfirmationGrace = 5 * time.Second
	OutputDrainTimeout    = 5 * time.Second
)
