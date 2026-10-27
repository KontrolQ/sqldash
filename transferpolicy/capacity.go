package transferpolicy

import (
	"context"
	"sync"

	"sqldash/config"
)

var capacity = struct {
	sync.Mutex
	active  int
	changed chan struct{}
}{changed: make(chan struct{})}

func Acquire(ctx context.Context) (func(), error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		capacity.Lock()
		if config.Transfer.MaximumConcurrentJobs == 0 || capacity.active < config.Transfer.MaximumConcurrentJobs {
			capacity.active++
			capacity.Unlock()
			var released sync.Once
			return func() {
				released.Do(func() {
					capacity.Lock()
					capacity.active--
					close(capacity.changed)
					capacity.changed = make(chan struct{})
					capacity.Unlock()
				})
			}, nil
		}
		changed := capacity.changed
		capacity.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}
