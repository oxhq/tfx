package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/internal/share"
)

func testEntry() *share.Entry {
	return &share.Entry{
		Level:     share.LevelInfo,
		Message:   "hello \"world\"",
		Fields:    share.Fields{"user": "gara", "badge": "skip"},
		Timestamp: time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
		Caller:    &share.CallerInfo{File: "/tmp/example/file.go", Line: 42},
	}
}

func TestDefaultFileOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultFileOptions()
	if opts.Format != share.FormatText || opts.MaxSize == 0 || opts.Permissions == 0 {
		t.Fatalf("unexpected default file options: %+v", opts)
	}
}

func TestFileWriterWriteAndHelpers(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "logs", "app.log")
	opts := DefaultFileOptions()
	opts.Compress = false

	fw, err := NewFileWriter(filename, opts)
	if err != nil {
		t.Fatalf("NewFileWriter() error = %v", err)
	}
	t.Cleanup(func() {
		_ = fw.Close()
	})

	entry := testEntry()
	text := fw.formatText(entry)
	if !strings.Contains(text, "hello \"world\"") || !strings.Contains(text, "user=gara") {
		t.Fatalf("unexpected text format: %q", text)
	}

	jsonText := fw.formatJSON(entry)
	if !strings.Contains(jsonText, `"message":"hello \"world\""`) {
		t.Fatalf("unexpected json format: %q", jsonText)
	}
	if got := fw.escapeJSON("a\tb\nc"); got != "a\\tb\\nc" {
		t.Fatalf("unexpected escaped json: %q", got)
	}
	if got := fw.shortFilename("/tmp/example/file.go"); got != "file.go" {
		t.Fatalf("unexpected short filename: %q", got)
	}

	if err := fw.Write(entry); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	// #nosec G304 -- filename is created inside the test temp directory.
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(contents), "hello \"world\"") {
		t.Fatalf("expected text log content, got %q", string(contents))
	}

	fw.options.Format = share.FormatJSON
	if err := fw.Write(entry); err != nil {
		t.Fatalf("Write() json error = %v", err)
	}
	// #nosec G304 -- filename is created inside the test temp directory.
	contents, err = os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() after json error = %v", err)
	}
	if !strings.Contains(string(contents), `"timestamp":"2026-03-22T12:00:00Z"`) {
		t.Fatalf("expected json log content, got %q", string(contents))
	}

	if err := fw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := fw.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestFileWriterRotationAndCleanup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filename := filepath.Join(dir, "app.log")
	opts := DefaultFileOptions()
	opts.MaxSize = 1
	opts.MaxBackups = 1
	opts.MaxAge = 0
	opts.Compress = false

	fw, err := NewFileWriter(filename, opts)
	if err != nil {
		t.Fatalf("NewFileWriter() error = %v", err)
	}
	t.Cleanup(func() {
		_ = fw.Close()
	})

	if !fw.needsRotation(2) {
		t.Fatal("expected tiny writer to require rotation")
	}
	backupName := fw.generateBackupName()
	if !strings.HasPrefix(backupName, filepath.Join(dir, "app.")) ||
		!strings.HasSuffix(backupName, ".log") {
		t.Fatalf("unexpected backup filename: %q", backupName)
	}

	if err := fw.Write(testEntry()); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "app.*.log"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(backups) == 0 {
		t.Fatal("expected at least one rotated backup file")
	}

	extraA := filepath.Join(dir, "app.older-a.log")
	extraB := filepath.Join(dir, "app.older-b.log")
	for _, name := range []string{extraA, extraB} {
		if err := os.WriteFile(name, []byte("old"), 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}
	if err := os.Chtimes(
		extraA,
		time.Now().Add(-3*time.Hour),
		time.Now().Add(-3*time.Hour),
	); err != nil {
		t.Fatalf("Chtimes(extraA) error = %v", err)
	}
	if err := os.Chtimes(
		extraB,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-2*time.Hour),
	); err != nil {
		t.Fatalf("Chtimes(extraB) error = %v", err)
	}

	fw.cleanup()
	backups, err = filepath.Glob(filepath.Join(dir, "app.*.log"))
	if err != nil {
		t.Fatalf("Glob() after cleanup error = %v", err)
	}
	if len(backups) > opts.MaxBackups {
		t.Fatalf("expected at most %d backup after cleanup, got %d", opts.MaxBackups, len(backups))
	}

	fw.compressFile(filename)
}
