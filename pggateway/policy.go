package pggateway

import (
	"context"
	"net"
)

type callbackResult[Value any] struct {
	value Value
	error error
}

func callBounded[Value any](ctx context.Context, slots chan struct{}, callback func() (Value, error)) (Value, error) {
	var empty Value
	if ctx.Err() != nil {
		return empty, ctx.Err()
	}
	if !acquireSlot(slots) {
		return empty, ErrConnectionLimit
	}
	completed := make(chan callbackResult[Value], 1)
	go func() {
		var result callbackResult[Value]
		returned := false
		defer func() {
			_ = recover()
			if !returned {
				result.error = ErrCallback
			}
			<-slots
			completed <- result
		}()
		result.value, result.error = callback()
		returned = true
	}()
	select {
	case <-ctx.Done():
		return empty, ctx.Err()
	case result := <-completed:
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}
		return result.value, result.error
	}
}

func (self *gateway) authorize(ctx context.Context, startup startupIdentity, peerIP net.IP) (*Identity, error) {
	validationContext, cancel := context.WithTimeout(ctx, self.options.ValidationTimeout)
	defer cancel()
	return callBounded(validationContext, self.policyCalls, func() (*Identity, error) {
		identity, authorizeError := self.options.Authorize(validationContext, startup.username, startup.database, append(net.IP(nil), peerIP...))
		if authorizeError != nil || identity == nil {
			return nil, ErrAccessDenied
		}
		bound := *identity
		bound.SourceIP = peerIP.String()
		if bound.CredentialID == 0 || bound.DatabaseID == 0 || bound.AccountID == 0 || bound.RoleName != startup.username || bound.DatabaseName != startup.database {
			return nil, ErrAccessDenied
		}
		checked := bound
		if validationError := self.options.Validate(validationContext, &checked); validationError != nil {
			return nil, ErrAccessDenied
		}
		return &bound, nil
	})
}

func sameAuthority(left, right Identity) bool {
	left.SourceIP, right.SourceIP = "", ""
	return left == right
}

func (self *session) validate() error {
	identity, validationError := self.gateway.authorize(self.context, self.startup, self.peerIP)
	if validationError != nil || identity == nil || *identity != self.identity {
		return ErrAccessDenied
	}
	return nil
}
