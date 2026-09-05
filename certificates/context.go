package certificates

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func shutdownAware() context.Context {
	held, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	go func() {
		<-held.Done()
		stop()
	}()

	return held
}

func quiet() *zap.Logger {
	return zap.NewNop()
}
