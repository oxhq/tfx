package writer

import (
	"sync"
	"testing"
	"time"

	"github.com/oxhq/tfx/internal/share"
)

type blockingWriter struct {
	started chan struct{}
	release chan struct{}

	mu     sync.Mutex
	writes int
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (w *blockingWriter) Write(entry *share.Entry) error {
	w.mu.Lock()
	w.writes++
	count := w.writes
	if count == 1 {
		close(w.started)
	}
	w.mu.Unlock()

	if count == 1 {
		<-w.release
	}

	return nil
}

func (w *blockingWriter) Close() error {
	return nil
}

func (w *blockingWriter) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writes
}

func TestAsyncWriterFlushWaitsForQueuedEntries(t *testing.T) {
	underlying := newBlockingWriter()
	aw := NewAsyncWriter(underlying, 1)

	if err := aw.Write(&share.Entry{Message: "flush"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	<-underlying.started

	flushed := make(chan struct{})
	go func() {
		aw.Flush()
		close(flushed)
	}()

	select {
	case <-flushed:
		t.Fatal("Flush returned before the queued entry was processed")
	case <-time.After(50 * time.Millisecond):
	}

	close(underlying.release)

	select {
	case <-flushed:
	case <-time.After(time.Second):
		t.Fatal("Flush did not return after the queued entry was processed")
	}

	if got := underlying.count(); got != 1 {
		t.Fatalf("Expected 1 processed write, got %d", got)
	}

	if err := aw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestAsyncWriterCloseDoesNotPanicWhenWriteIsBlocked(t *testing.T) {
	underlying := newBlockingWriter()
	aw := NewAsyncWriter(underlying, 1)

	if err := aw.Write(&share.Entry{Message: "first"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	<-underlying.started

	if err := aw.Write(&share.Entry{Message: "second"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	thirdDone := make(chan error, 1)
	go func() {
		thirdDone <- aw.Write(&share.Entry{Message: "third"})
	}()

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- aw.Close()
	}()

	select {
	case err := <-closeDone:
		t.Fatalf("Close() returned before the blocked entry could drain: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(underlying.release)

	select {
	case err := <-thirdDone:
		if err != nil {
			t.Fatalf("third Write() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Blocked Write() did not return")
	}

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close() did not return")
	}

	if got := underlying.count(); got != 2 {
		t.Fatalf("Expected 2 processed writes, got %d", got)
	}
}
