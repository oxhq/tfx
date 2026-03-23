package terminal

import (
	"bytes"
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"golang.org/x/term"
)

func ttyForTests(t *testing.T) *os.File {
	t.Helper()

	if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		t.Cleanup(func() { _ = tty.Close() })
		return tty
	}

	if IsTerminal(os.Stdout) {
		return os.Stdout
	}

	return nil
}

func TestDetectorDetectCapabilitiesBranches(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := NewDetector(&bytes.Buffer{}).detectCapabilities(); got != ModeNoColor {
		t.Fatalf("expected NO_COLOR to force ModeNoColor, got %v", got)
	}

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "24bit")
	if got := NewDetector(&bytes.Buffer{}).detectCapabilities(); got != ModeNoColor {
		t.Fatalf("expected non-terminal output to remain ModeNoColor, got %v", got)
	}

	tty := ttyForTests(t)
	if tty == nil {
		t.Skip("no terminal file available for capability branch coverage")
	}

	t.Setenv("NO_COLOR", "")
	t.Setenv("CI", "")
	t.Setenv("FORCE_COLOR", "")

	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-256color")
	if got := NewDetector(tty).detectCapabilities(); got != ModeTrueColor {
		t.Fatalf("expected truecolor terminal detection, got %v", got)
	}

	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "screen-256color")
	if got := NewDetector(tty).detectCapabilities(); got != Mode256 {
		t.Fatalf("expected 256-color terminal detection, got %v", got)
	}

	t.Setenv("TERM", "xterm")
	if got := NewDetector(tty).detectCapabilities(); got != ModeANSI {
		t.Fatalf("expected ANSI terminal detection, got %v", got)
	}
}

func TestModeStringsAndFileSizeLookup(t *testing.T) {
	t.Parallel()

	expected := map[Mode]string{
		ModeNoColor:   "NoColor",
		ModeANSI:      "ANSI",
		Mode256:       "256Color",
		ModeTrueColor: "TrueColor",
	}
	for mode, want := range expected {
		if got := mode.String(); got != want {
			t.Fatalf("unexpected mode string for %v: got %q want %q", mode, got, want)
		}
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	if _, _, err := GetSizeForFile(writer); err == nil {
		t.Fatal("expected pipe-backed file size lookup to fail")
	}
}

func TestSignalHandlerHandlesResizeAndContextStop(t *testing.T) {
	sh := NewSignalHandler()
	resizeCh := make(chan struct{}, 1)
	stopCh := make(chan struct{}, 1)

	sh.OnResize(func() { resizeCh <- struct{}{} })
	sh.OnStop(func() { stopCh <- struct{}{} })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		sh.Listen(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatalf("failed to send SIGWINCH: %v", err)
	}

	select {
	case <-resizeCh:
	case <-time.After(time.Second):
		t.Fatal("expected resize callback after SIGWINCH")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("signal listener did not stop after context cancellation")
	}

	select {
	case <-stopCh:
	case <-time.After(time.Second):
		t.Fatal("expected stop callback on context cancellation")
	}
}

func TestRestoreTerminalAndIsTerminalBranches(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	if isTerminal(reader.Fd()) {
		t.Fatal("pipe reader should not be detected as a terminal")
	}

	if err := RestoreTerminal(^uintptr(0), &term.State{}); err == nil {
		t.Fatal("expected RestoreTerminal to fail for invalid fd")
	}
}

func TestIsColorDisabledBranches(t *testing.T) {
	for _, key := range []string{
		"NO_COLOR",
		"TERM",
		"CI",
		"FORCE_COLOR",
		"CONTINUOUS_INTEGRATION",
		"BUILD_NUMBER",
		"JENKINS_URL",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"TRAVIS",
		"CIRCLECI",
	} {
		t.Setenv(key, "")
	}

	if isColorDisabled() {
		t.Fatal("expected color to remain enabled by default")
	}

	t.Setenv("TERM", "dumb")
	if !isColorDisabled() {
		t.Fatal("expected dumb terminal to disable color")
	}

	t.Setenv("TERM", "")
	t.Setenv("CI", "1")
	if !isColorDisabled() {
		t.Fatal("expected CI to disable color without FORCE_COLOR")
	}

	t.Setenv("FORCE_COLOR", "1")
	if isColorDisabled() {
		t.Fatal("expected FORCE_COLOR to re-enable color in CI")
	}
}

func TestSignalHandlerStopStopsListener(t *testing.T) {
	sh := NewSignalHandler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		sh.Listen(ctx)
		close(done)
	}()

	sh.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("signal listener did not stop after Stop()")
	}
}
