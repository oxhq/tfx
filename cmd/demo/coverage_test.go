package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestMainRunsDemo(t *testing.T) {
	oldStdout := os.Stdout

	tempFile, err := os.CreateTemp(t.TempDir(), "demo-stdout-*.log")
	if err != nil {
		t.Fatalf("failed to create temp stdout file: %v", err)
	}
	defer func() {
		os.Stdout = oldStdout
		_ = tempFile.Close()
	}()

	os.Stdout = tempFile
	main()

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to rewind stdout file: %v", err)
	}
	data, err := io.ReadAll(tempFile)
	if err != nil {
		t.Fatalf("failed to read demo output: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "TFX demo: running formfx showcase") {
		t.Fatalf("expected main banner, got %q", output)
	}
	if !strings.Contains(output, "Starting main task...") || !strings.Contains(output, "Done.") {
		t.Fatalf("expected full demo output, got %q", output)
	}
}
