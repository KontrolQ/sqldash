package pggateway

import (
	"context"
	"errors"
)

func (self *Registry) add(current *session) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.sessions == nil {
		self.sessions = make(map[*session]struct{})
	}
	self.sessions[current] = struct{}{}
}

func (self *Registry) remove(current *session) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	delete(self.sessions, current)
}

func (self *Registry) TerminateForCredential(ctx context.Context, credentialID uint) error {
	if self == nil || ctx == nil || credentialID == 0 {
		return ErrInvalidOptions
	}
	self.mutex.Lock()
	sessions := make([]*session, 0)
	for current := range self.sessions {
		if current.identity.CredentialID == credentialID {
			sessions = append(sessions, current)
		}
	}
	self.mutex.Unlock()
	for _, current := range sessions {
		go current.stop()
	}
	var terminationErrors []error
	for _, current := range sessions {
		select {
		case <-ctx.Done():
			return errors.Join(append(terminationErrors, ctx.Err())...)
		case <-current.done:
			if current.stopError != nil {
				terminationErrors = append(terminationErrors, current.stopError)
			}
		}
	}
	return errors.Join(terminationErrors...)
}
