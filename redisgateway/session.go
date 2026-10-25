package redisgateway

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"time"

	"sqldash/rediswire"
)

func (self *session) run(parent context.Context) {
	select {
	case self.server.handshakes <- struct{}{}:
	case <-parent.Done():
		return
	case <-self.done:
		return
	default:
		_ = self.client.SetWriteDeadline(time.Now().Add(time.Second))
		_ = rediswire.WriteAll(self.client, []byte(connectionLimit))
		return
	}
	acquired := true
	defer func() {
		if acquired {
			<-self.server.handshakes
		}
	}()
	client, downstream, upstream, err := self.authenticate(parent)
	<-self.server.handshakes
	acquired = false
	if err != nil {
		return
	}
	self.proxy(parent, client, downstream, upstream)
}

func (self *session) inspect(parent context.Context, arguments [][]byte) error {
	if len(arguments) == 0 {
		return RebindingDenied
	}
	name := arguments[0]
	if !bytes.EqualFold(name, []byte("AUTH")) && !bytes.EqualFold(name, []byte("HELLO")) && !bytes.EqualFold(name, []byte("SELECT")) {
		return nil
	}
	validation, cancel := context.WithTimeout(parent, self.server.options.ValidationTimeout)
	defer cancel()
	if err := self.validate(validation); err != nil {
		self.enforce(parent)
		return AccessDenied
	}
	if bytes.EqualFold(name, []byte("SELECT")) {
		if len(arguments) != 2 || !bytes.Equal(arguments[1], []byte("0")) {
			return RebindingDenied
		}
		return nil
	}
	username, err := authenticationUsername(arguments, bytes.EqualFold(name, []byte("AUTH")))
	if err != nil || username != "" && username != self.binding.Username {
		return RebindingDenied
	}
	return nil
}

func (self *session) enforce(parent context.Context) {
	self.enforcement.Do(func() {
		self.close()
		termination, cancel := context.WithTimeout(context.WithoutCancel(parent), self.server.options.TerminationTimeout)
		err := self.terminate(termination)
		cancel()
		if err != nil {
			unsafeContext, unsafeCancel := context.WithTimeout(context.WithoutCancel(parent), self.server.options.TerminationTimeout)
			enforcementError := self.unsafe(unsafeContext, err)
			unsafeCancel()
			if enforcementError != nil {
				self.server.fail(EnforcementFailed)
			}
		}
	})
}

func (self *session) proxy(parent context.Context, client net.Conn, downstream, upstream *bufio.Reader) {
	frontendFinished := make(chan struct{})
	backendFinished := make(chan struct{})
	go func() {
		defer close(frontendFinished)
		for {
			if err := rediswire.RelayCommand(downstream, self.backend, rediswire.RelayOptions{Inspect: func(arguments [][]byte) error { return self.inspect(parent, arguments) }}); err != nil {
				return
			}
		}
	}()
	go func() {
		defer close(backendFinished)
		_, _ = io.Copy(client, upstream)
	}()
	defer func() { self.close(); <-frontendFinished; <-backendFinished }()
	ticker := time.NewTicker(self.server.options.ValidationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-parent.Done():
			return
		case <-self.done:
			return
		case <-frontendFinished:
			return
		case <-backendFinished:
			return
		case <-ticker.C:
			validation, cancel := context.WithTimeout(parent, self.server.options.ValidationTimeout)
			err := self.validate(validation)
			cancel()
			if err != nil {
				self.enforce(parent)
				return
			}
		}
	}
}
