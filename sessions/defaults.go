package sessions

import "time"

const (
	LogPrefix = "Sessions"

	AccountKey  = "account"
	ProblemKey  = "flash-problem"
	DoneKey     = "flash-done"
	IntendedKey = "intended-path"
	Lifetime    = 720 * time.Hour
)
