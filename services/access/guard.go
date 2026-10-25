package access

import (
	"context"
	"errors"
	"sync"
	"time"
)

func Guard(parent context.Context, check func(context.Context) error, enforce func(context.Context) error) (context.Context, func(), error) {
	if check == nil {
		return parent, nil, Refused
	}
	verify := func(ctx context.Context) error {
		bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return check(bounded)
	}
	if err := verify(parent); err != nil {
		return parent, nil, err
	}
	work, cancel := context.WithCancelCause(parent)
	done := make(chan struct{})
	finished := make(chan struct{})
	var once sync.Once
	fail := func(err error) {
		cancel(err)
		if enforce != nil {
			cleanup, stop := context.WithTimeout(context.WithoutCancel(parent), 4*time.Second)
			defer stop()
			if enforcementError := enforce(cleanup); enforcementError != nil {
				cancel(errors.Join(err, enforcementError))
			}
		}
	}
	go func() {
		defer close(finished)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-work.Done():
				fail(context.Cause(work))
				return
			case <-ticker.C:
				if err := verify(work); err != nil {
					fail(err)
					return
				}
			}
		}
	}()
	stop := func() { once.Do(func() { close(done) }); <-finished; cancel(context.Canceled) }
	return work, stop, nil
}
