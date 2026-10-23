package supervisor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"sqldash/utils/logger"
)

func New(processes ...Process) *Supervisor {
	supervisor := &Supervisor{}
	for _, process := range processes {
		supervisor.processes = append(supervisor.processes, &managedProcess{
			process:  process,
			finished: make(chan struct{}),
			health:   ProcessHealth{Name: process.Name},
		})
	}
	return supervisor
}

func Run(processes ...Process) {
	runContext, stopEverything := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopEverything()
	if err := New(processes...).Run(runContext); err != nil {
		logger.Errorf(LogPrefix, "%v", err)
	}
}

func (self *Supervisor) Run(ctx context.Context) error {
	self.mutex.Lock()
	if self.running {
		self.mutex.Unlock()
		return errors.New("supervisor has already started")
	}
	seenNames := make(map[string]bool)
	for _, process := range self.processes {
		if process.process.Name == "" || seenNames[process.process.Name] {
			self.mutex.Unlock()
			return errors.New("supervised processes need unique nonempty names")
		}
		seenNames[process.process.Name] = true
	}
	self.running = true
	for _, process := range self.processes {
		processContext, cancel := context.WithCancel(context.WithoutCancel(ctx))
		process.cancel = cancel
		go self.keepAlive(processContext, process)
	}
	self.mutex.Unlock()
	logger.Infof(LogPrefix, SupervisingLog, len(self.processes))
	<-ctx.Done()
	self.shutdown()
	logger.Infof(LogPrefix, AllStoppedLog)
	return nil
}

func (self *Supervisor) Health() []ProcessHealth {
	self.mutex.RLock()
	defer self.mutex.RUnlock()
	health := make([]ProcessHealth, 0, len(self.processes))
	for _, process := range self.processes {
		health = append(health, process.health)
	}
	return health
}

func (self *Supervisor) Stop(name string) error {
	self.mutex.RLock()
	var target *managedProcess
	for _, process := range self.processes {
		if process.process.Name == name {
			target = process
			break
		}
	}
	if target == nil || target.cancel == nil {
		self.mutex.RUnlock()
		return errors.New("supervised process is unavailable")
	}
	cancel := target.cancel
	self.mutex.RUnlock()
	cancel()
	grace := target.process.StopGrace
	if grace <= 0 {
		grace = ShutdownGrace
	}
	timer := time.NewTimer(grace + StopConfirmationGrace)
	defer timer.Stop()
	select {
	case <-target.finished:
		return nil
	case <-timer.C:
		return errors.New("supervised process shutdown could not be confirmed")
	}
}

func (self *Supervisor) shutdown() {
	ordered := append([]*managedProcess(nil), self.processes...)
	sort.SliceStable(ordered, func(first, second int) bool {
		return ordered[first].process.ShutdownGroup < ordered[second].process.ShutdownGroup
	})
	for first := 0; first < len(ordered); {
		last := first
		for last < len(ordered) && ordered[last].process.ShutdownGroup == ordered[first].process.ShutdownGroup {
			ordered[last].cancel()
			last++
		}
		for _, process := range ordered[first:last] {
			<-process.finished
		}
		first = last
	}
}

func (self *Supervisor) update(process *managedProcess, update func(*ProcessHealth)) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	update(&process.health)
}

func (self *Supervisor) failed(process *managedProcess, err error) {
	if err == nil {
		err = errors.New("supervised process exited unexpectedly")
	}
	self.update(process, func(health *ProcessHealth) {
		health.Running = false
		health.Ready = false
		health.Failures++
		health.LastFailure = err.Error()
	})
	if process.process.OnFailure != nil {
		process.process.OnFailure(err)
	}
}

func (self *Supervisor) keepAlive(ctx context.Context, process *managedProcess) {
	defer close(process.finished)
	defer self.update(process, func(health *ProcessHealth) { health.Running = false; health.Ready = false })
	if process.process.Prepare != nil {
		if err := process.process.Prepare(ctx); err != nil {
			if ctx.Err() == nil {
				self.failed(process, err)
				logger.Errorf(LogPrefix, StartFailedLog, process.process.Name, err)
			}
			return
		}
	}
	failures := 0
	for ctx.Err() == nil {
		startedAt := time.Now()
		err := self.runOnce(ctx, process)
		if ctx.Err() != nil {
			logger.Infof(LogPrefix, StoppingLog, process.process.Name)
			return
		}
		self.failed(process, err)
		logger.Warnf(LogPrefix, ExitedLog, process.process.Name, err)
		self.mutex.RLock()
		readyAt := process.readyAt
		self.mutex.RUnlock()
		if !readyAt.Before(startedAt) && time.Since(readyAt) >= HealthyRunDuration {
			failures = 0
		} else {
			failures++
		}
		delay := backoffFor(failures)
		logger.Infof(LogPrefix, RestartingLog, process.process.Name, delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (self *Supervisor) runOnce(ctx context.Context, process *managedProcess) error {
	configuration := process.process
	logger.Infof(LogPrefix, StartingLog, configuration.Name)
	command := exec.Command(configuration.Path, configuration.Arguments...)
	command.Env = append(os.Environ(), configuration.Environment...)
	if configuration.IsolatedEnvironment {
		command.Env = append([]string{}, configuration.Environment...)
	}
	command.Dir = configuration.Directory
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := configureCredentials(command, configuration.Credentials); err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		logger.Errorf(LogPrefix, StartFailedLog, configuration.Name, err)
		return err
	}
	self.update(process, func(health *ProcessHealth) {
		health.Running = true
		health.Ready = configuration.Readiness == nil
		if health.Ready {
			process.readyAt = time.Now()
		}
	})
	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()
	var waitError error
	defer func() {
		self.update(process, func(health *ProcessHealth) { health.Running = false; health.Ready = false })
		if configuration.OnStopped != nil {
			configuration.OnStopped(waitError)
		}
	}()
	if configuration.Readiness != nil {
		readinessTimeout := configuration.ReadyTimeout
		if readinessTimeout <= 0 {
			readinessTimeout = ReadinessTimeout
		}
		readinessContext, cancel := context.WithTimeout(ctx, readinessTimeout)
		ready := make(chan error, 1)
		go func() { ready <- awaitReadiness(readinessContext, configuration.Readiness) }()
		select {
		case waitError = <-finished:
			cancel()
			return waitError
		case readinessError := <-ready:
			cancel()
			if readinessError != nil {
				waitError = stopAndWait(command, configuration, finished)
				return errors.New("authenticated process readiness failed")
			}
			self.update(process, func(health *ProcessHealth) {
				health.Ready = true
				health.Failures = 0
				health.LastFailure = ""
				process.readyAt = time.Now()
			})
		case <-readinessContext.Done():
			cancel()
			waitError = stopAndWait(command, configuration, finished)
			return errors.New("authenticated process readiness timed out")
		}
	}
	select {
	case waitError = <-finished:
		return waitError
	case <-ctx.Done():
		waitError = stopAndWait(command, configuration, finished)
		return waitError
	}
}

func awaitReadiness(ctx context.Context, readiness func(context.Context) error) error {
	for ctx.Err() == nil {
		if err := readiness(ctx); err == nil {
			return ctx.Err()
		}
		timer := time.NewTimer(ReadinessInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	return ctx.Err()
}

func backoffFor(failures int) time.Duration {
	if failures > 20 {
		return MaximumRestartDelay
	}
	delay := RestartDelay << max(failures, 0)
	if delay > MaximumRestartDelay || delay <= 0 {
		return MaximumRestartDelay
	}
	return delay
}
