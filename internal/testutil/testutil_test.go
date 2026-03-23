package testutil

import "testing"

func TestSafeBufferWriteStringAndReset(t *testing.T) {
	t.Parallel()

	var buf SafeBuffer
	if _, err := buf.Write([]byte("hello")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if got := buf.String(); got != "hello" {
		t.Fatalf("unexpected buffer contents: %q", got)
	}

	buf.Reset()
	if got := buf.String(); got != "" {
		t.Fatalf("expected reset buffer to be empty, got %q", got)
	}
}
