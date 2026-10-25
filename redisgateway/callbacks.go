package redisgateway

import (
	"context"
	"net"
)

type callbackResult[Value any] struct {
	value Value
	err   error
}

func invoke[Value any](ctx context.Context, slots chan struct{}, fallback error, callback func() (Value, error), dispose ...func(Value)) (Value, error) {
	var empty Value
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	select {
	case slots <- struct{}{}:
	case <-ctx.Done():
		return empty, ctx.Err()
	default:
		return empty, fallback
	}
	finished := make(chan callbackResult[Value])
	go func() {
		defer func() { <-slots }()
		result := callbackResult[Value]{err: fallback}
		func() {
			defer func() {
				if recover() != nil {
					result.value = empty
					result.err = fallback
				}
			}()
			result.value, result.err = callback()
		}()
		select {
		case finished <- result:
		case <-ctx.Done():
			if len(dispose) > 0 {
				dispose[0](result.value)
			}
		}
	}()
	select {
	case <-ctx.Done():
		return empty, ctx.Err()
	case result := <-finished:
		if err := ctx.Err(); err != nil {
			if len(dispose) > 0 {
				dispose[0](result.value)
			}
			return empty, err
		}
		return result.value, result.err
	}
}

func (self *session) authorize(ctx context.Context, username string) (*Binding, error) {
	return invoke(ctx, self.server.callbacks, AccessDenied, func() (*Binding, error) { return self.server.options.Authorize(ctx, username, self.peer) })
}

func (self *session) validate(ctx context.Context) error {
	_, err := invoke(ctx, self.server.callbacks, AccessDenied, func() (struct{}, error) {
		return struct{}{}, self.server.options.Validate(ctx, self.binding, self.peer)
	})
	return err
}

func (self *session) dial(ctx context.Context) (net.Conn, error) {
	return invoke(ctx, self.server.callbacks, BackendUnavailable, func() (net.Conn, error) {
		connection, err := self.server.options.Dial(ctx, self.binding)
		if ctx.Err() != nil && connection != nil {
			connection.Close()
			return nil, ctx.Err()
		}
		return connection, err
	}, func(connection net.Conn) {
		if connection != nil {
			connection.Close()
		}
	})
}

func (self *session) terminate(ctx context.Context) error {
	_, err := invoke(ctx, self.server.terminations, EnforcementFailed, func() (struct{}, error) { return struct{}{}, self.server.options.Terminate(ctx, self.binding) })
	return err
}

func (self *session) unsafe(ctx context.Context, failure error) error {
	_, err := invoke(ctx, self.server.enforcements, EnforcementFailed, func() (struct{}, error) {
		self.server.options.Unsafe(ctx, self.binding, failure)
		return struct{}{}, nil
	})
	return err
}
