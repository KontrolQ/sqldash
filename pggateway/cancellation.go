package pggateway

import (
	"context"
	"crypto/subtle"
	"encoding/binary"
	"net"
	"sync"
	"time"
)

type requestRate struct {
	mutex   sync.Mutex
	started time.Time
	count   int
}

func (self *requestRate) allow(maximum int) bool {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	now := time.Now()
	if now.Sub(self.started) >= time.Second {
		self.started = now
		self.count = 0
	}
	if self.count >= maximum {
		return false
	}
	self.count++
	return true
}

func (self *gateway) forwardCancel(frontend net.Conn, packet startupPacket, peerIP net.IP) {
	if len(packet.contents) < 12+minimumCancelKeyBytes || len(packet.contents) > maximumCancelPacketBytes || !self.cancelRate.allow(self.options.MaximumCancelRequests) {
		return
	}
	backendPID := binary.BigEndian.Uint32(packet.contents[8:12])
	key := packet.contents[12:]
	defer clear(key)
	self.mutex.Lock()
	current := self.backends[backendPID]
	self.mutex.Unlock()
	if current == nil || !current.matchesCancel(key) || !current.cancelRate.allow(self.options.MaximumSessionCancels) {
		return
	}
	ctx, cancel := context.WithDeadline(current.context, packet.deadline)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if deadlineError := frontend.SetDeadline(deadline); deadlineError != nil {
		return
	}
	identity, authorizeError := self.authorize(ctx, current.startup, peerIP)
	if authorizeError != nil || identity == nil || !sameAuthority(*identity, current.identity) {
		return
	}
	dialer := net.Dialer{Timeout: self.options.BackendConnectTimeout}
	backend, dialError := dialer.DialContext(ctx, "tcp", self.options.BackendAddress)
	if dialError != nil {
		return
	}
	defer closeConnection(backend)
	if deadlineError := backend.SetDeadline(deadline); deadlineError != nil {
		return
	}
	if !current.attachCancel(backend, key) {
		return
	}
	defer current.removeCancel(backend)
	if writeError := writeAll(backend, packet.contents); writeError != nil {
		return
	}
	var response [1]byte
	_, _ = backend.Read(response[:])
}

func (self *session) matchesCancel(key []byte) bool {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.ready && !self.stopped && subtle.ConstantTimeCompare(self.cancelKey, key) == 1
}

func (self *session) attachCancel(connection net.Conn, key []byte) bool {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if !self.ready || self.stopped || self.context.Err() != nil || subtle.ConstantTimeCompare(self.cancelKey, key) != 1 {
		return false
	}
	if self.cancellations == nil {
		self.cancellations = make(map[net.Conn]struct{})
	}
	self.cancellations[connection] = struct{}{}
	return true
}

func (self *session) removeCancel(connection net.Conn) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	delete(self.cancellations, connection)
}
