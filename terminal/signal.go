package terminal

import (
	"context"
	"sync"
)

// SignalHandler defines the interface for handling terminal signals
type SignalHandler struct {
	onResize func()
	onStop   func()
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler() *SignalHandler {
	return &SignalHandler{
		stopCh: make(chan struct{}),
	}
}

// OnResize sets the callback for resize events
func (sh *SignalHandler) OnResize(callback func()) {
	sh.onResize = callback
}

// OnStop sets the callback for stop signals
func (sh *SignalHandler) OnStop(callback func()) {
	sh.onStop = callback
}

// Listen starts listening for signals
func (sh *SignalHandler) Listen(ctx context.Context) {
	listenForSignals(ctx, sh)
}

// Stop stops the signal handler
func (sh *SignalHandler) Stop() {
	sh.stopOnce.Do(func() {
		close(sh.stopCh)
	})
}
