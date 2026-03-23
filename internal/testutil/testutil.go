package testutil

import (
	"bytes"
	"sync"
)

// SafeBuffer is a thread-safe wrapper for bytes.Buffer
// to avoid data races in concurrent tests
type SafeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (s *SafeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *SafeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *SafeBuffer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
}
