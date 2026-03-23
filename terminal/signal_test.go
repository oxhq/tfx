package terminal

import (
	"context"
	"testing"
	"time"
)

func TestSignalHandlerStopIsIdempotent(t *testing.T) {
	sh := NewSignalHandler()
	sh.Stop()
	sh.Stop()
}

func TestSignalHandlerListenStopsOnStop(t *testing.T) {
	sh := NewSignalHandler()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		sh.Listen(ctx)
		close(done)
	}()

	sh.Stop()
	sh.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("signal listener did not stop after Stop")
	}
}
