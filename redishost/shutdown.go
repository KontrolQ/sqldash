package redishost

import (
	"errors"
	"sync"
	"syscall"
)

func (self *manager) shutdown() error {
	self.ready.Store(false)
	self.stopping.Store(true)
	_ = self.server.Close()
	self.mutex.Lock()
	pending := make([]*managedInstance, 0, len(self.processes))
	for _, current := range self.processes {
		pending = append(pending, current)
	}
	self.processes = make(map[uint]*managedInstance)
	self.mutex.Unlock()
	for _, current := range pending {
		_ = current.command.Process.Signal(syscall.SIGTERM)
	}
	var waiting sync.WaitGroup
	failures := make(chan error, len(pending))
	for _, current := range pending {
		waiting.Go(func() {
			if err := stopProcess(current); err != nil {
				failures <- err
			}
		})
	}
	waiting.Wait()
	close(failures)
	var collected []error
	for failure := range failures {
		collected = append(collected, failure)
	}
	return errors.Join(collected...)
}
