package terminal

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestModeStringUnknownAndCallbacks(t *testing.T) {
	t.Parallel()

	if got := Mode(99).String(); got != "Unknown" {
		t.Fatalf("unexpected mode string: %q", got)
	}

	sh := NewSignalHandler()
	resized := false
	stopped := false
	sh.OnResize(func() { resized = true })
	sh.OnStop(func() { stopped = true })
	if sh.onResize == nil || sh.onStop == nil {
		t.Fatal("expected callbacks to be registered")
	}
	sh.onResize()
	sh.onStop()
	if !resized || !stopped {
		t.Fatalf("expected callbacks to run, got resized=%v stopped=%v", resized, stopped)
	}
}

func TestColorDisabledScenarios(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !isColorDisabled() {
		t.Fatal("expected NO_COLOR to disable color")
	}

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if !isColorDisabled() {
		t.Fatal("expected TERM=dumb to disable color")
	}

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("CI", "1")
	t.Setenv("FORCE_COLOR", "1")
	if isColorDisabled() {
		t.Fatal("expected FORCE_COLOR to override CI no-color default")
	}
}

func TestPlatformHelpersWithInvalidInputs(t *testing.T) {
	t.Parallel()

	if IsTerminal(&bytes.Buffer{}) {
		t.Fatal("buffer should not be a terminal")
	}
	if _, _, err := GetSizeForFile(nil); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("expected invalid file error, got %v", err)
	}
	if _, _, err := GetSizeForFD(^uintptr(0)); err == nil {
		t.Fatal("expected invalid file descriptor error")
	}
	if _, err := MakeRaw(^uintptr(0)); err == nil {
		t.Fatal("expected raw mode failure for invalid fd")
	}
	_, _, _ = GetSize()
}
