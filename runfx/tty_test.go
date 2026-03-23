package runfx

import (
	"bytes"
	"testing"

	"github.com/oxhq/tfx/writer"
)

func TestDetectTTY(t *testing.T) {
	info := DetectTTY()

	// Basic checks - these will depend on test environment
	if info.IsTTY {
		// If it's a TTY, should have some color support
		if info.NoColor && info.ANSI {
			t.Error("Inconsistent color detection: NoColor is true but ANSI is also true")
		}
	} else {
		// If not a TTY, should not have color support
		if !info.NoColor {
			t.Error("Non-TTY should have NoColor=true")
		}
	}
}

func TestDetectTTYForOutput(t *testing.T) {
	// Test with a buffer (definitely not a TTY)
	buf := &bytes.Buffer{}
	info := DetectTTYForOutput(buf)

	if info.IsTTY {
		t.Error("Buffer should not be detected as TTY")
	}

	if !info.NoColor {
		t.Error("Buffer should have NoColor=true")
	}

	if info.ANSI {
		t.Error("Buffer should not support ANSI")
	}

	if info.TrueColor {
		t.Error("Buffer should not support TrueColor")
	}
}

func TestTTYInfoConsistency(t *testing.T) {
	info := DetectTTY()

	// Test logical consistency
	if info.TrueColor && info.NoColor {
		t.Error("Cannot have both TrueColor and NoColor")
	}

	if info.ANSI && info.NoColor {
		t.Error("Cannot have both ANSI and NoColor")
	}

	if info.TrueColor && !info.ANSI {
		t.Error("TrueColor support should imply ANSI support")
	}
}

func TestFallbackOutput(t *testing.T) {
	// This test mainly checks that FallbackOutput doesn't panic
	// The actual output depends on whether we're in a TTY or not
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FallbackOutput panicked: %v", r)
		}
	}()

	FallbackOutput("test message")
}

func TestTTYDetectionWithTerminalWriter(t *testing.T) {
	// Test that our TTY detection is consistent with TerminalWriter
	buf := &bytes.Buffer{}

	// Create TerminalWriter directly
	termWriter := writer.NewTerminalWriter(buf, writer.TerminalOptions{})

	// Get our TTY info
	info := DetectTTYForOutput(buf)

	// Compare results
	if info.IsTTY != termWriter.IsTerminal() {
		t.Error("TTY detection inconsistent between TTYInfo and TerminalWriter")
	}

	if info.ANSI != termWriter.SupportsColor() {
		t.Error("Color support detection inconsistent between TTYInfo and TerminalWriter")
	}
}
