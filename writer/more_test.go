package writer

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/terminal"
)

type failingShareWriter struct {
	err        error
	writeCount int
}

func (w *failingShareWriter) Write(*share.Entry) error {
	w.writeCount++
	return w.err
}

func (w *failingShareWriter) Close() error { return nil }

type failingByteWriter struct {
	err error
}

func (w failingByteWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestAsyncWriterHelpersAndErrorChannel(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("write failed")
	underlying := &failingShareWriter{err: wantErr}
	aw := NewAsyncWriter(underlying, 1)
	t.Cleanup(func() {
		_ = aw.Close()
	})

	if aw.UnderlyingWriter() != underlying {
		t.Fatal("expected UnderlyingWriter to expose wrapped writer")
	}
	if aw.Errors() == nil {
		t.Fatal("expected Errors channel to be non-nil")
	}

	if err := aw.Write(&share.Entry{Message: "boom"}); err != nil {
		t.Fatalf("unexpected async write error: %v", err)
	}

	select {
	case err := <-aw.Errors():
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected write error %v, got %v", wantErr, err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected async writer to publish underlying error")
	}
}

func TestConsoleWriterUpdateOptionsAndTerminalWriterModes(t *testing.T) {
	t.Parallel()

	first := &bytes.Buffer{}
	second := &bytes.Buffer{}
	console := NewConsoleWriter(first, ConsoleOptions{BadgeWidth: 4})
	console.UpdateOptions(second, ConsoleOptions{
		Level:        share.LevelWarn,
		Format:       share.FormatText,
		BadgeWidth:   9,
		Theme:        color.NordTheme,
		DisableColor: true,
	})

	if console.output != second || console.badgeWidth != 9 {
		t.Fatalf("unexpected console writer state after update: %+v", console)
	}
	if console.options.Level != share.LevelWarn || console.options.Format != share.FormatText {
		t.Fatalf("expected console options to be replaced, got %+v", console.options)
	}

	entry := &share.Entry{
		Level:     share.LevelWarn,
		Message:   "updated",
		Timestamp: time.Now(),
	}
	if err := console.Write(entry); err != nil {
		t.Fatalf("unexpected console write error: %v", err)
	}
	if !bytes.Contains(second.Bytes(), []byte("updated")) {
		t.Fatalf("expected write to use updated output buffer, got %q", second.String())
	}

	terminalWriter := NewTerminalWriter(&bytes.Buffer{}, TerminalOptions{ForceColor: true})
	if got := terminalWriter.GetColorMode(); got != color.ModeTrueColor {
		t.Fatalf("expected forced color mode to be truecolor, got %v", got)
	}

	terminalWriter = NewTerminalWriter(&bytes.Buffer{}, TerminalOptions{DisableColor: true})
	if got := terminalWriter.GetColorMode(); got != color.ModeNoColor {
		t.Fatalf("expected disabled color mode to be no-color, got %v", got)
	}

	terminalWriter = NewTerminalWriter(&bytes.Buffer{}, TerminalOptions{})
	terminalWriter.detector.ForceMode(terminal.Mode256)
	if got := terminalWriter.GetColorMode(); got != color.Mode256Color {
		t.Fatalf("expected detector Mode256 to map to 256-color, got %v", got)
	}
	terminalWriter.detector.ForceMode(terminal.ModeANSI)
	if got := terminalWriter.GetColorMode(); got != color.ModeANSI {
		t.Fatalf("expected detector ModeANSI to map to ANSI, got %v", got)
	}
}

func TestTerminalWriterAndFileCompressionHelpers(t *testing.T) {
	t.Parallel()

	writer := NewTerminalWriter(&bytes.Buffer{}, TerminalOptions{})
	if _, err := writer.EnableRawMode(); err == nil {
		t.Fatal("expected raw mode to fail on non-file writer")
	}
	if err := writer.RestoreMode(nil); err == nil {
		t.Fatal("expected restore mode to fail on non-file writer")
	}

	reader, file, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() {
		_ = reader.Close()
		_ = file.Close()
	}()

	withTTYFile := NewTerminalWriter(&bytes.Buffer{}, TerminalOptions{TTYFile: file})
	if _, _, err := withTTYFile.GetSize(); err == nil {
		t.Fatal("expected invalid pipe size lookup to fail")
	}

	dir := t.TempDir()
	filename := filepath.Join(dir, "compress.log")
	if err := os.WriteFile(filename, []byte("log line"), 0o600); err != nil {
		t.Fatalf("failed to seed file for compression test: %v", err)
	}

	fw := &FileWriter{filename: filename, options: DefaultFileOptions()}
	fw.compressFile(filename)
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("expected placeholder compression to leave file intact, got %v", err)
	}
}

func TestTerminalWriterNoopControlsAndBufferedWrites(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	writer := NewTerminalWriter(&out, TerminalOptions{DoubleBuffer: true})
	if writer.IsTerminal() {
		t.Fatal("expected buffer-backed terminal writer to be non-tty")
	}

	if err := writer.Clear(); err != nil {
		t.Fatalf("expected clear to be a no-op on non-tty writer, got %v", err)
	}
	if err := writer.MoveCursor(2, 4); err != nil {
		t.Fatalf("expected move cursor to be a no-op on non-tty writer, got %v", err)
	}
	if err := writer.HideCursor(); err != nil {
		t.Fatalf("expected hide cursor to be a no-op on non-tty writer, got %v", err)
	}
	if err := writer.ShowCursor(); err != nil {
		t.Fatalf("expected show cursor to be a no-op on non-tty writer, got %v", err)
	}

	n, err := writer.Write([]byte("frame"))
	if err != nil || n != len("frame") {
		t.Fatalf("unexpected first buffered write result: n=%d err=%v", n, err)
	}
	if got := out.String(); got != "frame" {
		t.Fatalf("unexpected first buffered write output: %q", got)
	}

	n, err = writer.Write([]byte("frame"))
	if err != nil || n != len("frame") {
		t.Fatalf("unexpected repeated buffered write result: n=%d err=%v", n, err)
	}
	if got := out.String(); got != "frame" {
		t.Fatalf("expected repeated frame to be suppressed, got %q", got)
	}

	wantErr := errors.New("buffered write failed")
	failing := NewTerminalWriter(
		failingByteWriter{err: wantErr},
		TerminalOptions{DoubleBuffer: true},
	)
	if _, err := failing.Write([]byte("boom")); !errors.Is(err, wantErr) {
		t.Fatalf("expected buffered write error %v, got %v", wantErr, err)
	}
}

func TestConsoleWriterFormattingHelpers(t *testing.T) {
	t.Parallel()

	styled := NewConsoleWriter(&bytes.Buffer{}, ConsoleOptions{ForceColor: true, BadgeWidth: 8})
	if got := styled.formatBadgeTag(&share.Entry{
		Level:  share.LevelInfo,
		Fields: share.Fields{"badge_styled": "<styled>"},
	}); got != "<styled>" {
		t.Fatalf("expected pre-styled badge to pass through, got %q", got)
	}

	multiBadge := styled.formatBadgeTag(&share.Entry{
		Level: share.LevelSuccess,
		Fields: share.Fields{
			"badge":    "SHIP READY",
			"bg_color": color.ModernBlue,
		},
	})
	if !strings.Contains(multiBadge, "\x1b[") {
		t.Fatalf("expected colored multi-word badge, got %q", multiBadge)
	}

	plain := NewConsoleWriter(&bytes.Buffer{}, ConsoleOptions{
		DisableColor: true,
		Timestamp:    true,
		TimeFormat:   "15:04:05",
		ShowCaller:   true,
		BadgeWidth:   8,
	})
	entry := &share.Entry{
		Level:     share.LevelWarn,
		Message:   "danger",
		IndentStr: "  ",
		Timestamp: time.Date(2026, 3, 22, 12, 34, 56, 0, time.UTC),
		Caller:    &share.CallerInfo{File: "/tmp/example/app.go", Line: 42},
		Fields: share.Fields{
			"badge":       "Deploy",
			"badge_color": color.ModernRed,
			"region":      "mx",
			"type":        "success",
		},
	}

	rendered := plain.formatBadge(entry)
	for _, want := range []string{"  ", "[12:34:56]", "app.go:42", "danger", "region=mx"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in badge output, got %q", want, rendered)
		}
	}

	fallbackBadge := plain.formatBadgeTag(&share.Entry{Level: share.LevelWarn})
	if !strings.Contains(fallbackBadge, "Warn") {
		t.Fatalf("expected warn fallback badge, got %q", fallbackBadge)
	}

	fields := styled.formatFields(share.Fields{
		"key":          "value",
		"badge":        "skip",
		"badge_color":  color.ModernRed,
		"badge_styled": "skip",
		"bg_color":     color.ModernBlue,
	})
	if !strings.Contains(fields, "value") || strings.Contains(fields, "skip") {
		t.Fatalf("unexpected formatted fields output: %q", fields)
	}

	if got := plain.shortFilename("plain.go"); got != "plain.go" {
		t.Fatalf("unexpected short filename without path: %q", got)
	}
}
