package pggateway

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type gateway struct {
	options          Options
	context          context.Context
	cancel           context.CancelFunc
	listener         net.Listener
	mutex            sync.Mutex
	connections      map[net.Conn]struct{}
	backends         map[uint32]*session
	failure          error
	unsafeOnce       sync.Once
	workers          sync.WaitGroup
	connectionSlots  chan struct{}
	sessionSlots     chan struct{}
	handshakeSlots   chan struct{}
	policyCalls      chan struct{}
	terminationCalls chan struct{}
	cancelRate       requestRate
}

func Run(ctx context.Context, options Options) error {
	normalized, optionsError := options.normalized()
	if optionsError != nil {
		return optionsError
	}
	if ctx == nil {
		return ErrInvalidOptions
	}
	runContext, cancel := context.WithCancel(ctx)
	defer cancel()
	listener, listenError := (&net.ListenConfig{}).Listen(runContext, "tcp", normalized.ListenAddress)
	if listenError != nil {
		return listenError
	}
	self := &gateway{
		options: normalized, context: runContext, cancel: cancel, listener: listener,
		connections: make(map[net.Conn]struct{}), backends: make(map[uint32]*session),
		connectionSlots:  quotaSlots(normalized.MaximumConnections),
		sessionSlots:     quotaSlots(normalized.MaximumSessions),
		handshakeSlots:   make(chan struct{}, normalized.MaximumHandshakes),
		policyCalls:      make(chan struct{}, max(normalized.MaximumConnections, DefaultMaximumPolicyCalls)),
		terminationCalls: make(chan struct{}, max(normalized.MaximumSessions, DefaultMaximumTerminationCalls)),
	}
	shutdownDone := make(chan struct{})
	go func() {
		<-runContext.Done()
		self.closeIngress()
		close(shutdownDone)
	}()
	var acceptError error
	for {
		connection, connectionError := listener.Accept()
		if connectionError != nil {
			if runContext.Err() == nil {
				acceptError = connectionError
			}
			break
		}
		if !acquireSlot(self.connectionSlots) {
			closeConnection(connection)
			continue
		}
		self.mutex.Lock()
		if runContext.Err() != nil {
			self.mutex.Unlock()
			releaseSlot(self.connectionSlots)
			closeConnection(connection)
			break
		}
		self.connections[connection] = struct{}{}
		self.mutex.Unlock()
		self.workers.Add(1)
		go self.handle(connection)
	}
	cancel()
	<-shutdownDone
	self.workers.Wait()
	self.mutex.Lock()
	failure := self.failure
	self.mutex.Unlock()
	return errors.Join(acceptError, failure)
}

func (self *gateway) handle(connection net.Conn) {
	defer self.workers.Done()
	defer func() {
		closeConnection(connection)
		self.mutex.Lock()
		delete(self.connections, connection)
		self.mutex.Unlock()
		releaseSlot(self.connectionSlots)
	}()
	if !acquireSlot(self.handshakeSlots) {
		return
	}
	handshakeHeld := true
	defer func() {
		if handshakeHeld {
			<-self.handshakeSlots
		}
	}()
	frontend, packet, negotiationError := self.negotiate(connection)
	if negotiationError != nil {
		if errors.Is(negotiationError, ErrTLSRequired) {
			writeProtocolError(frontend, "28000", tlsRequiredMessage)
		} else if errors.Is(negotiationError, ErrInvalidStartup) {
			writeProtocolError(frontend, "08P01", invalidStartupMessage)
		}
		return
	}
	peerIP := connectionIP(connection)
	if peerIP == nil {
		return
	}
	if packet.code == cancelRequestCode {
		self.forwardCancel(frontend, packet, peerIP)
		return
	}
	startup, startupError := parseStartup(packet)
	if startupError != nil {
		message := invalidStartupMessage
		if errors.Is(startupError, ErrUnsupportedProtocol) {
			message = unsupportedProtocolMessage
		}
		writeProtocolError(frontend, "08P01", message)
		return
	}
	if !acquireSlot(self.sessionSlots) {
		writeProtocolError(frontend, "53300", connectionLimitMessage)
		return
	}
	defer releaseSlot(self.sessionSlots)
	identity, authorizeError := self.authorize(self.context, startup, peerIP)
	if authorizeError != nil {
		writeProtocolError(frontend, "28000", accessDeniedMessage)
		return
	}
	current := &session{
		gateway: self, frontend: frontend, identity: *identity, startup: startup,
		peerIP: append(net.IP(nil), peerIP...), protocol: startup.protocol, done: make(chan struct{}),
	}
	current.context, current.cancel = context.WithCancel(self.context)
	self.options.Registry.add(current)
	defer self.options.Registry.remove(current)
	defer current.stop()
	go current.watchPolicy()
	if validationError := current.validate(); validationError != nil {
		writeProtocolError(frontend, "28000", accessDeniedMessage)
		current.stop()
		return
	}
	dialer := net.Dialer{Timeout: self.options.BackendConnectTimeout}
	backend, dialError := dialer.DialContext(current.context, "tcp", self.options.BackendAddress)
	if dialError != nil {
		writeProtocolError(frontend, "08006", backendUnavailableMessage)
		return
	}
	if attachError := current.attachBackend(backend); attachError != nil {
		return
	}
	if deadlineError := backend.SetDeadline(time.Now().Add(self.options.HandshakeTimeout)); deadlineError != nil {
		return
	}
	if writeError := current.startBackend(packet.contents); writeError != nil {
		return
	}
	if authenticationError := current.authenticate(); authenticationError != nil {
		message := handshakeFailedMessage
		if errors.Is(authenticationError, ErrChannelBinding) {
			message = channelBindingMessage
		}
		writeProtocolError(frontend, "28000", message)
		return
	}
	if validationError := current.validate(); validationError != nil {
		current.stop()
		return
	}
	if deadlineError := frontend.SetDeadline(time.Time{}); deadlineError != nil {
		return
	}
	if deadlineError := backend.SetDeadline(time.Time{}); deadlineError != nil {
		return
	}
	<-self.handshakeSlots
	handshakeHeld = false
	copied := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(backend, frontend)
		copied <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(frontend, backend)
		copied <- struct{}{}
	}()
	<-copied
	current.stop()
	<-copied
}

func (self *gateway) unsafe(terminationError error) {
	self.unsafeOnce.Do(func() {
		failure := errors.Join(ErrUnsafe, terminationError)
		self.mutex.Lock()
		self.failure = failure
		self.mutex.Unlock()
		self.cancel()
		self.closeIngress()
		self.options.Unsafe(failure)
	})
}

func (self *gateway) closeIngress() {
	_ = self.listener.Close()
	self.mutex.Lock()
	connections := make([]net.Conn, 0, len(self.connections))
	for connection := range self.connections {
		connections = append(connections, connection)
	}
	self.mutex.Unlock()
	for _, connection := range connections {
		closeConnection(connection)
	}
}

func quotaSlots(maximum int) chan struct{} {
	if maximum == 0 {
		return nil
	}
	return make(chan struct{}, maximum)
}

func releaseSlot(slots chan struct{}) {
	if slots != nil {
		<-slots
	}
}

func acquireSlot(slots chan struct{}) bool {
	if slots == nil {
		return true
	}
	select {
	case slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func connectionIP(connection net.Conn) net.IP {
	host, _, addressError := net.SplitHostPort(connection.RemoteAddr().String())
	if addressError != nil {
		return nil
	}
	return net.ParseIP(host)
}

func closeConnection(connection net.Conn) {
	if connection != nil {
		_ = connection.SetDeadline(time.Now())
		_ = connection.Close()
	}
}
