package redistransfer

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"sqldash/models"
	"sqldash/redishost"
	"sqldash/transferpolicy"
)

type Options struct {
	Authorize func(context.Context) error
	Timeout   time.Duration
}

type authorizedWriter struct {
	context   context.Context
	writer    io.Writer
	authorize func(context.Context) error
}

func (self *authorizedWriter) Write(contents []byte) (int, error) {
	if err := self.context.Err(); err != nil {
		return 0, err
	}
	if err := self.authorize(self.context); err != nil {
		return 0, err
	}
	return self.writer.Write(contents)
}

type authorizedReader struct {
	context   context.Context
	reader    io.Reader
	authorize func(context.Context) error
}

func (self *authorizedReader) Read(contents []byte) (int, error) {
	if err := self.context.Err(); err != nil {
		return 0, err
	}
	if err := self.authorize(self.context); err != nil {
		return 0, err
	}
	return self.reader.Read(contents)
}

func transfer(ctx context.Context, options Options, operation func(context.Context, func(context.Context) error) error) error {
	if options.Authorize == nil || options.Timeout < 0 {
		return redishost.PolicyUnconfirmed
	}
	var authority sync.Mutex
	authorize := func(current context.Context) error {
		authority.Lock()
		defer authority.Unlock()
		if err := current.Err(); err != nil {
			return err
		}
		checked, stop := context.WithTimeout(current, 2*time.Second)
		defer stop()
		if err := options.Authorize(checked); err != nil {
			return err
		}
		return checked.Err()
	}
	bounded, stop := transferpolicy.Context(ctx, options.Timeout)
	defer stop()
	release, capacityError := transferpolicy.Acquire(bounded)
	if capacityError != nil {
		return capacityError
	}
	defer release()
	work, cancel := context.WithCancelCause(bounded)
	defer cancel(nil)
	if err := authorize(work); err != nil {
		return err
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-work.Done():
				return
			case <-ticker.C:
				if err := authorize(work); err != nil {
					cancel(err)
					return
				}
			}
		}
	}()
	result := operation(work, authorize)
	close(done)
	<-finished
	if err := context.Cause(work); err != nil {
		return errors.Join(result, err)
	}
	if result != nil {
		return result
	}
	return authorize(work)
}

func ExportWithOptions(ctx context.Context, source *models.Database, writer io.Writer, options Options) error {
	if writer == nil {
		return redishost.InvalidSettings
	}
	return transfer(ctx, options, func(work context.Context, authorize func(context.Context) error) error {
		return redishost.Snapshot(work, source, &authorizedWriter{context: work, writer: writer, authorize: authorize})
	})
}

func ImportWithOptions(ctx context.Context, target *models.Database, path string, options Options) error {
	return transfer(ctx, options, func(work context.Context, authorize func(context.Context) error) error {
		file, err := redishost.OpenSnapshot(path)
		if err != nil {
			return err
		}
		defer file.Close()
		return redishost.Restore(work, target, &authorizedReader{context: work, reader: file, authorize: authorize})
	})
}

func CloneWithOptions(ctx context.Context, source, target *models.Database, options Options) error {
	if source == nil || target == nil || source.ID == target.ID {
		return redishost.IdentityMismatch
	}
	return transfer(ctx, options, func(work context.Context, authorize func(context.Context) error) error {
		return redishost.Clone(work, source, target)
	})
}

func Cleanup(ctx context.Context, target *models.Database) error {
	return redishost.CleanupClone(ctx, target)
}
