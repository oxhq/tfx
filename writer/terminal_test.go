package writer

import (
	"bytes"
	"strings"
	"testing"
)

func TestTerminalWriterBasic(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := TerminalOptions{
		DoubleBuffer: false,
		ForceColor:   false,
		DisableColor: false,
	}

	tw := NewTerminalWriter(buf, opts)

	// Test basic write
	data := []byte("test data")
	n, err := tw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned wrong count: got %d, want %d", n, len(data))
	}

	if buf.String() != "test data" {
		t.Errorf("Output incorrect: got %q, want %q", buf.String(), "test data")
	}
}

func TestTerminalWriterDoubleBuffer(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := TerminalOptions{
		DoubleBuffer: true,
		ForceColor:   false,
		DisableColor: false,
	}

	tw := NewTerminalWriter(buf, opts)

	// First write
	data1 := []byte("first")
	_, _ = tw.Write(data1)

	// Same write again - should not output due to double buffering
	_, _ = tw.Write(data1)

	// Different write - should output
	data2 := []byte("second")
	_, _ = tw.Write(data2)

	output := buf.String()
	if strings.Count(output, "first") > 1 {
		t.Error("Double buffering failed - same content written multiple times")
	}

	if !strings.Contains(output, "second") {
		t.Error("Second write not found in output")
	}
}

func TestTerminalWriterColorDetection(t *testing.T) {
	buf := &bytes.Buffer{}

	// Test with color forced
	optsForced := TerminalOptions{ForceColor: true}
	twForced := NewTerminalWriter(buf, optsForced)
	if !twForced.SupportsColor() {
		t.Error("Should support color when forced")
	}

	// Test with color disabled
	optsDisabled := TerminalOptions{DisableColor: true}
	twDisabled := NewTerminalWriter(buf, optsDisabled)
	if twDisabled.SupportsColor() {
		t.Error("Should not support color when disabled")
	}
}

func TestTerminalWriterIsTerminal(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := TerminalOptions{}
	tw := NewTerminalWriter(buf, opts)

	// Buffer is not a terminal
	if tw.IsTerminal() {
		t.Error("Buffer should not be detected as terminal")
	}
}

func TestTerminalWriterCursorOperations(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := TerminalOptions{}
	tw := NewTerminalWriter(buf, opts)

	// These operations should not panic for non-terminal
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Cursor operations panicked: %v", r)
		}
	}()

	err := tw.HideCursor()
	if err != nil {
		t.Errorf("HideCursor failed: %v", err)
	}

	err = tw.ShowCursor()
	if err != nil {
		t.Errorf("ShowCursor failed: %v", err)
	}

	err = tw.MoveCursor(10, 20)
	if err != nil {
		t.Errorf("MoveCursor failed: %v", err)
	}

	// For non-terminal, these should be no-ops, so buffer should be empty
	if buf.Len() > 0 {
		t.Errorf(
			"Cursor operations should be no-ops for non-terminal, but got output: %q",
			buf.String(),
		)
	}
}

func TestTerminalWriterClear(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := TerminalOptions{DoubleBuffer: true}
	tw := NewTerminalWriter(buf, opts)

	// Write some data
	_, _ = tw.Write([]byte("test"))

	// Clear should not panic
	err := tw.Clear()
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}

	// For non-terminal, clear should be no-op
	// But it should reset the buffer
	_, _ = tw.Write([]byte("test"))
	_, _ = tw.Write([]byte("test")) // Same content - should not output due to buffering

	// After clear and rewrite, should output again
	_ = tw.Clear()
	_, _ = tw.Write([]byte("test"))

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Error("Expected output after clear and rewrite")
	}
}

func TestTerminalWriterGetSize(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := TerminalOptions{}
	tw := NewTerminalWriter(buf, opts)

	// GetSize should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetSize panicked: %v", r)
		}
	}()

	width, height, err := tw.GetSize()
	// In test environment, this might fail, which is okay
	if err == nil {
		if width <= 0 || height <= 0 {
			t.Errorf("Invalid terminal size: %dx%d", width, height)
		}
	}
}

func TestTerminalWriterCloseFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := TerminalOptions{}
	tw := NewTerminalWriter(buf, opts)

	// These should not panic and should return nil
	err := tw.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	err = tw.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
