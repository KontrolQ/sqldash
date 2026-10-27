package supervisor

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"

	"sqldash/utils/logger"
)

type Credentials struct {
	UserID  uint32
	GroupID uint32
}

func (self *Credentials) Configure(command *exec.Cmd) error {
	return configureCredentials(command, self)
}

type Process struct {
	Name                string
	Path                string
	Arguments           []string
	Environment         []string
	IsolatedEnvironment bool
	Directory           string
	LogPrefix           string
	LogFormat           logger.OutputFormat
	Credentials         *Credentials
	Prepare             func(context.Context) error
	Readiness           func(context.Context) error
	ReadyTimeout        time.Duration
	StopSignal          os.Signal
	StopGrace           time.Duration
	ShutdownGroup       int
	OnFailure           func(error)
	OnStopped           func(error)
}

type ProcessHealth struct {
	Name        string
	Running     bool
	Ready       bool
	Failures    int
	LastFailure string
}

type managedProcess struct {
	process  Process
	cancel   context.CancelFunc
	finished chan struct{}
	health   ProcessHealth
	readyAt  time.Time
}

type Supervisor struct {
	mutex     sync.RWMutex
	processes []*managedProcess
	running   bool
}
