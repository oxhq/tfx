package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRunFormFXDemoWithOptions(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	runFormFXDemoWithOptions(&out, func(time.Duration) {})

	rendered := out.String()
	if !strings.Contains(rendered, "Starting main task...") {
		t.Fatalf("expected intro text, got %q", rendered)
	}
	if !strings.Contains(rendered, "Task completed.") {
		t.Fatalf("expected completion text, got %q", rendered)
	}
	if !strings.Contains(rendered, "Done.") {
		t.Fatalf("expected spinner completion text, got %q", rendered)
	}
}
