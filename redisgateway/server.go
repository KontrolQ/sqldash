package redisgateway

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"time"

	"sqldash/rediswire"
)

func Run(ctx context.Context, supplied Options) error {
	options, err := normalize(supplied)
	if err != nil {
		return err
	}
	listener := options.Listener
	if listener == nil {
		listener, err = (&net.ListenConfig{}).Listen(ctx, "tcp", options.ListenAddress)
		if err != nil {
			return err
		}
	}
	defer listener.Close()
	self := &gateway{options: options, sessions: make(map[*session]struct{}), databases: make(map[uint]int), handshakes: make(chan struct{}, options.MaximumHandshakes), callbacks: make(chan struct{}, max(options.MaximumConnections, DefaultMaximumCallbacks)+options.MaximumHandshakes), terminations: make(chan struct{}, 8), enforcements: make(chan struct{}, 16), failures: make(chan error, 1), listener: listener}
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			listener.Close()
			self.closeSessions()
		case <-stopped:
		}
	}()
	defer func() { self.closeSessions(); self.waiting.Wait() }()
	for {
		connection, err := listener.Accept()
		if err != nil {
			select {
			case failure := <-self.failures:
				return failure
			default:
			}
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		self.mutex.Lock()
		if options.MaximumConnections > 0 && len(self.sessions) >= options.MaximumConnections {
			self.mutex.Unlock()
			_ = connection.SetWriteDeadline(time.Now().Add(time.Second))
			_ = rediswire.WriteAll(connection, []byte(connectionLimit))
			connection.Close()
			continue
		}
		current := &session{server: self, client: connection, done: make(chan struct{})}
		if address, err := netip.ParseAddrPort(connection.RemoteAddr().String()); err == nil {
			current.peer = address.Addr().Unmap()
		}
		self.sessions[current] = struct{}{}
		self.waiting.Add(1)
		self.mutex.Unlock()
		go func() {
			defer self.waiting.Done()
			defer self.release(current)
			defer current.close()
			defer func() {
				if recover() != nil {
					current.close()
				}
			}()
			current.run(ctx)
		}()
	}
}

func (self *gateway) fail(err error) {
	select {
	case self.failures <- err:
	default:
	}
	self.listener.Close()
	self.closeSessions()
}

func (self *gateway) closeSessions() {
	self.mutex.Lock()
	pending := make([]*session, 0, len(self.sessions))
	for current := range self.sessions {
		pending = append(pending, current)
	}
	self.mutex.Unlock()
	for _, current := range pending {
		current.close()
	}
}

func (self *gateway) reserve(current *session, databaseID uint) bool {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.options.MaximumSessionsPerDatabase > 0 && self.databases[databaseID] >= self.options.MaximumSessionsPerDatabase {
		return false
	}
	self.databases[databaseID]++
	current.databaseID = databaseID
	return true
}

func (self *gateway) release(current *session) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	delete(self.sessions, current)
	if current.databaseID != 0 {
		self.databases[current.databaseID]--
		if self.databases[current.databaseID] == 0 {
			delete(self.databases, current.databaseID)
		}
	}
}

func (self *session) close() {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.closed {
		return
	}
	self.closed = true
	close(self.done)
	self.client.Close()
	if self.backend != nil {
		self.backend.Close()
	}
}

func (self *session) setBackend(connection net.Conn) bool {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.closed {
		connection.Close()
		return false
	}
	self.backend = connection
	return true
}
