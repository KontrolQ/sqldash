package supervisor

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"sqldash/utils/logger"
)

var waitGroup sync.WaitGroup

func Run(processes ...Process) {
	runContext, stopEverything := context.WithCancel(context.Background())
	defer stopEverything()

	logger.Infof(LogPrefix, SupervisingLog, len(processes))

	for _, process := range processes {
		waitGroup.Add(1)
		go keepAlive(runContext, process)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	received := <-signals
	logger.Infof(LogPrefix, SignalLog, received)

	stopEverything()
	waitGroup.Wait()

	logger.Infof(LogPrefix, AllStoppedLog)
}

func keepAlive(runContext context.Context, process Process) {
	defer waitGroup.Done()

	held := attempt{process: process}

	for {
		if runContext.Err() != nil {
			return
		}

		held.startedAt = time.Now()
		runError := runOnce(runContext, process)

		if runContext.Err() != nil {
			logger.Infof(LogPrefix, StoppingLog, process.Name)
			return
		}

		logger.Warnf(LogPrefix, ExitedLog, process.Name, runError)

		if time.Since(held.startedAt) >= HealthyRunDuration {
			held.failures = 0
		} else {
			held.failures++
		}

		delay := backoffFor(held.failures)
		logger.Infof(LogPrefix, RestartingLog, process.Name, delay)

		select {
		case <-runContext.Done():
			return
		case <-time.After(delay):
		}
	}
}

func runOnce(runContext context.Context, process Process) error {
	logger.Infof(LogPrefix, StartingLog, process.Name)

	command := exec.Command(process.Path, process.Arguments...)
	command.Env = append(os.Environ(), process.Environment...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if startError := command.Start(); startError != nil {
		logger.Errorf(LogPrefix, StartFailedLog, process.Name, startError)
		return startError
	}

	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()

	select {
	case waitError := <-finished:
		return waitError
	case <-runContext.Done():
		terminate(command)

		select {
		case waitError := <-finished:
			return waitError
		case <-time.After(ShutdownGrace):
			command.Process.Kill()
			return <-finished
		}
	}
}

func backoffFor(failures int) time.Duration {
	delay := RestartDelay << failures

	if delay > MaximumRestartDelay || delay <= 0 {
		return MaximumRestartDelay
	}

	return delay
}
