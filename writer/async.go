package writer

import (
	"sync"

	"github.com/oxhq/tfx/internal/share"
)

// AsyncWriter provides an asynchronous, buffered writer decorator.
// It wraps another writer and performs writes in a separate goroutine.
type AsyncWriter struct {
	underlyingWriter share.Writer
	logCh            chan *share.Entry
	errCh            chan error
	doneCh           chan struct{}
	mu               sync.Mutex
	cond             *sync.Cond
	pending          int
	closed           bool
	runWg            sync.WaitGroup
}

// NewAsyncWriter creates a new asynchronous writer.
func NewAsyncWriter(underlying share.Writer, bufferSize int) *AsyncWriter {
	aw := &AsyncWriter{
		underlyingWriter: underlying,
		logCh:            make(chan *share.Entry, bufferSize),
		errCh:            make(chan error, 1),
		doneCh:           make(chan struct{}),
	}
	aw.cond = sync.NewCond(&aw.mu)

	aw.runWg.Add(1)
	go aw.run()

	return aw
}

// Write sends a log entry to the buffer.
// This method is non-blocking.
func (aw *AsyncWriter) Write(entry *share.Entry) error {
	aw.mu.Lock()
	if aw.closed {
		aw.mu.Unlock()
		return nil
	}
	aw.pending++
	doneCh := aw.doneCh
	logCh := aw.logCh
	aw.mu.Unlock()

	select {
	case logCh <- entry:
		return nil
	case <-doneCh:
		aw.finishPending()
		return nil
	}
}

// Close flushes the buffer and stops the writer.
func (aw *AsyncWriter) Close() error {
	aw.mu.Lock()
	if aw.closed {
		aw.mu.Unlock()
		return nil
	}
	aw.closed = true
	close(aw.doneCh)
	for aw.pending > 0 {
		aw.cond.Wait()
	}
	close(aw.logCh)
	aw.mu.Unlock()

	aw.runWg.Wait()
	return aw.underlyingWriter.Close()
}

// Errors returns a channel for receiving write errors.
func (aw *AsyncWriter) Errors() <-chan error {
	return aw.errCh
}

// Flush waits for all buffered messages to be written.
func (aw *AsyncWriter) Flush() {
	aw.mu.Lock()
	for aw.pending > 0 {
		aw.cond.Wait()
	}
	aw.mu.Unlock()

	if flusher, ok := aw.underlyingWriter.(interface{ Flush() }); ok {
		flusher.Flush()
	}
}

// run is the background goroutine that performs writes.
func (aw *AsyncWriter) run() {
	defer aw.runWg.Done()
	for entry := range aw.logCh {
		if err := aw.underlyingWriter.Write(entry); err != nil {
			select {
			case aw.errCh <- err:
			default:
				// Error channel is full, drop the error
			}
		}
		aw.finishPending()
	}
}

func (aw *AsyncWriter) finishPending() {
	aw.mu.Lock()
	if aw.pending > 0 {
		aw.pending--
		if aw.pending == 0 {
			aw.cond.Broadcast()
		}
	}
	aw.mu.Unlock()
}

// UnderlyingWriter returns the wrapped writer.
func (aw *AsyncWriter) UnderlyingWriter() share.Writer {
	return aw.underlyingWriter
}
