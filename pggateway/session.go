package pggateway

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

type session struct {
	gateway       *gateway
	frontend      net.Conn
	backend       net.Conn
	cancellations map[net.Conn]struct{}
	context       context.Context
	cancel        context.CancelFunc
	identity      Identity
	startup       startupIdentity
	peerIP        net.IP
	mutex         sync.Mutex
	backendPID    uint32
	cancelKey     []byte
	protocol      uint32
	ready         bool
	stopped       bool
	stopOnce      sync.Once
	stopError     error
	done          chan struct{}
	cancelRate    requestRate
}

func (self *session) attachBackend(backend net.Conn) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.stopped || self.context.Err() != nil {
		closeConnection(backend)
		return ErrAccessDenied
	}
	self.backend = backend
	return nil
}

func (self *session) startBackend(contents []byte) error {
	self.mutex.Lock()
	if self.stopped {
		self.mutex.Unlock()
		return ErrAccessDenied
	}
	backend := self.backend
	self.mutex.Unlock()
	return writeAll(backend, contents)
}

func (self *session) watchPolicy() {
	ticker := time.NewTicker(self.gateway.options.ValidationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-self.context.Done():
			if self.gateway.context.Err() != nil {
				self.stop()
			}
			return
		case <-ticker.C:
			if validationError := self.validate(); validationError != nil {
				self.stop()
				return
			}
		}
	}
}

func (self *session) stop() {
	self.stopOnce.Do(func() {
		self.mutex.Lock()
		self.stopped = true
		self.ready = false
		backend, backendPID := self.backend, self.backendPID
		cancellations := make([]net.Conn, 0, len(self.cancellations))
		for connection := range self.cancellations {
			cancellations = append(cancellations, connection)
		}
		clear(self.cancelKey)
		self.cancelKey = nil
		self.mutex.Unlock()
		self.cancel()
		closeConnection(self.frontend)
		closeConnection(backend)
		for _, connection := range cancellations {
			closeConnection(connection)
		}
		self.gateway.mutex.Lock()
		if self.gateway.backends[backendPID] == self {
			delete(self.gateway.backends, backendPID)
		}
		self.gateway.mutex.Unlock()
		defer close(self.done)
		if backendPID == 0 {
			return
		}
		terminationContext, cancel := context.WithTimeout(context.Background(), self.gateway.options.TerminationTimeout)
		defer cancel()
		identity := self.identity
		_, terminationError := callBounded(terminationContext, self.gateway.terminationCalls, func() (struct{}, error) {
			return struct{}{}, self.gateway.options.Terminate(terminationContext, &identity, backendPID)
		})
		if terminationError != nil {
			self.stopError = errors.Join(ErrUnsafe, terminationError)
			self.gateway.unsafe(self.stopError)
		}
	})
}
