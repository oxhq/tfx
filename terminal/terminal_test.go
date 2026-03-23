package terminal

import (
	"bytes"
	"io"
	"testing"
)

func TestNewDetectorAndMode(t *testing.T) {
	buf := &bytes.Buffer{}
	det := NewDetector(buf)
	if det == nil {
		t.Fatal("NewDetector returned nil")
	}
	if det.GetMode().String() == "" {
		t.Error("GetMode returned empty string")
	}
}

func TestDetectorSetOutputAndForceMode(t *testing.T) {
	det := NewDetector(&bytes.Buffer{})
	buf2 := &bytes.Buffer{}
	det.SetOutput(buf2)
	det.ForceMode(ModeTrueColor)
	if det.GetMode() != ModeTrueColor {
		t.Error("ForceMode did not set mode to ModeTrueColor")
	}
}

func TestDetectorCapabilities(t *testing.T) {
	det := NewDetector(&bytes.Buffer{})
	_ = det.SupportsColor()
	_ = det.SupportsANSI()
	_ = det.Supports256Color()
	_ = det.SupportsTrueColor()
	_ = det.GetOS()
}

func TestIsTerminalAndEnableANSI(t *testing.T) {
	// Platform-specific: just check that the functions are callable
	_ = IsTerminal(io.Discard)
	_ = TryEnableANSI()
}

func TestIsColorDisabled(t *testing.T) {
	_ = isColorDisabled()
}
