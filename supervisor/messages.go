package supervisor

const (
	StartingLog    = "Starting %s"
	StartFailedLog = "Failed to start %s: %v"
	ExitedLog      = "%s exited: %v"
	RestartingLog  = "Restarting %s in %s"
	StoppingLog    = "Stopping %s"
	AllStoppedLog  = "Every process has stopped"
	SignalLog      = "Received %s, shutting down"
	SupervisingLog = "Supervising %d processes"
)
