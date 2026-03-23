package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeArtifactPath(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	path, err := NormalizeArtifactPath(baseDir, "dist/app.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantAbsolute := filepath.Join(baseDir, "dist", "app.bin")
	if path.Absolute != wantAbsolute {
		t.Fatalf("expected absolute path %q, got %q", wantAbsolute, path.Absolute)
	}
	if path.Relative != filepath.Join("dist", "app.bin") {
		t.Fatalf(
			"expected relative path %q, got %q",
			filepath.Join("dist", "app.bin"),
			path.Relative,
		)
	}
	if got := path.Display(); got != path.Relative {
		t.Fatalf("expected compact display path %q, got %q", path.Relative, got)
	}
}

func TestNormalizeArtifactDeclarationsDefaultsAndDedupes(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(baseDir, "dist"), 0o750); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	binaryPath := filepath.Join(baseDir, "dist", "app.bin")
	if err := os.WriteFile(binaryPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	records, err := NormalizeArtifactDeclarations(baseDir, []ArtifactDeclaration{
		{Path: "dist/app.bin", Description: "Primary binary"},
		{Path: "dist/app.bin", Name: "duplicate"},
		{Path: "dist/other.txt", Optional: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records after dedupe, got %d", len(records))
	}

	if records[0].Name != "app.bin" {
		t.Fatalf("expected default name from path, got %q", records[0].Name)
	}
	if records[0].Description != "Primary binary" {
		t.Fatalf("expected preserved description, got %q", records[0].Description)
	}
	if !records[0].Present {
		t.Fatal("expected existing file to be marked present")
	}
	if records[1].Name != "other.txt" {
		t.Fatalf("expected second record name from path, got %q", records[1].Name)
	}
	if !records[1].Optional {
		t.Fatal("expected optional declaration to be preserved")
	}
}

func TestBuildArtifactSummaryMergesDeclaredAndDiscoveredRecords(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	distDir := filepath.Join(baseDir, "dist")
	if err := os.MkdirAll(distDir, 0o750); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "app.bin"), []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to write app binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "log.txt"), []byte("trace"), 0o600); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	summary, err := BuildArtifactSummary(
		baseDir,
		"release",
		"Release flow",
		[]ArtifactDeclaration{
			{Path: "dist/app.bin", Name: "binary", Description: "Release binary"},
			{Path: "dist/missing.txt", Optional: true},
		},
		[]string{
			"dist/app.bin",
			"./dist/log.txt",
			"dist/log.txt",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.StepID != "release" {
		t.Fatalf("expected step id to be preserved, got %q", summary.StepID)
	}
	if summary.StepTitle != "Release flow" {
		t.Fatalf("expected step title to be preserved, got %q", summary.StepTitle)
	}
	if summary.Count() != 3 {
		t.Fatalf("expected 3 merged artifact records, got %d", summary.Count())
	}
	if summary.PresentCount() != 2 {
		t.Fatalf("expected 2 present artifacts, got %d", summary.PresentCount())
	}
	if summary.MissingCount() != 1 {
		t.Fatalf("expected 1 missing artifact, got %d", summary.MissingCount())
	}
	if summary.ProblemCount() != 0 {
		t.Fatalf("expected no artifact problems, got %d", summary.ProblemCount())
	}
	if summary.TotalBytes() != int64(len("data")+len("trace")) {
		t.Fatalf("unexpected total size: %d", summary.TotalBytes())
	}

	rows := summary.TableRows()
	if len(rows) != 3 {
		t.Fatalf("expected 3 table rows, got %d", len(rows))
	}
	if rows[0][0] != "binary" {
		t.Fatalf("expected declared name to win merge, got %q", rows[0][0])
	}
	if rows[0][2] != "declared, discovered" {
		t.Fatalf("expected merged source label, got %q", rows[0][2])
	}
	if rows[0][3] != "present" {
		t.Fatalf("expected present state, got %q", rows[0][3])
	}
	if rows[1][2] != "discovered" {
		t.Fatalf("expected discovered-only source label, got %q", rows[1][2])
	}
	if rows[2][3] != "missing" {
		t.Fatalf("expected missing state, got %q", rows[2][3])
	}

	brief := summary.Brief()
	for _, want := range []string{"3 artifact(s)", "2 present", "1 missing", "9 B"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("expected brief summary to contain %q, got %q", want, brief)
		}
	}
}

func TestFormatArtifactSize(t *testing.T) {
	t.Parallel()

	if got := FormatArtifactSize(0); got != "0 B" {
		t.Fatalf("expected zero bytes to stay compact, got %q", got)
	}
	if got := FormatArtifactSize(1536); got != "1.5 KiB" {
		t.Fatalf("expected kibibyte formatting, got %q", got)
	}
}
