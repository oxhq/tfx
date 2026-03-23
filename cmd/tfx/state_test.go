package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func restoreStateResolvers() func() {
	originalConfigDir := userConfigDir
	originalTempDir := tempDir

	return func() {
		userConfigDir = originalConfigDir
		tempDir = originalTempDir
	}
}

func TestDefaultStatePathUsesUserConfigDir(t *testing.T) {
	defer restoreStateResolvers()()

	configDir := filepath.Join(t.TempDir(), "config")
	userConfigDir = func() (string, error) {
		return configDir, nil
	}
	tempDir = func() string {
		t.Fatal("tempDir fallback should not be used when config dir is available")
		return ""
	}

	path, err := DefaultStatePath()
	if err != nil {
		t.Fatalf("unexpected path error: %v", err)
	}
	want := filepath.Join(configDir, "tfx", stateFileName)
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestDefaultStatePathFallsBackToTempDir(t *testing.T) {
	defer restoreStateResolvers()()

	tempRoot := filepath.Join(t.TempDir(), "tmp")
	userConfigDir = func() (string, error) {
		return "", errors.New("config unavailable")
	}
	tempDir = func() string {
		return tempRoot
	}

	path, err := DefaultStatePath()
	if err != nil {
		t.Fatalf("unexpected path error: %v", err)
	}
	want := filepath.Join(tempRoot, "tfx", stateFileName)
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestStateStoreSaveLoadRoundTrip(t *testing.T) {
	store := NewStateStore(filepath.Join(t.TempDir(), "nested", stateFileName))
	fixedNow := time.Date(2026, time.March, 23, 10, 30, 0, 0, time.UTC)
	store.Now = func() time.Time { return fixedNow }

	state := WrapperState{
		SelectedFlow: "release",
		SelectedLane: "production",
		ProjectName:  "demo",
		RecentRuns: []WrapperRun{
			{
				Flow:       "ci",
				Lane:       "preview",
				Project:    "demo",
				Status:     "ok",
				StartedAt:  fixedNow.Add(-time.Hour),
				FinishedAt: fixedNow.Add(-45 * time.Minute),
			},
		},
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	info, err := os.Stat(store.Path)
	if err != nil {
		t.Fatalf("expected state file to exist: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected state file permissions 0600, got %v", info.Mode().Perm())
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if loaded.Version != stateSchemaV1 {
		t.Fatalf("expected version %d, got %d", stateSchemaV1, loaded.Version)
	}
	if loaded.SelectedFlow != state.SelectedFlow ||
		loaded.SelectedLane != state.SelectedLane ||
		loaded.ProjectName != state.ProjectName {
		t.Fatalf("unexpected loaded selection: %#v", loaded)
	}
	if !loaded.UpdatedAt.Equal(fixedNow) {
		t.Fatalf("expected updated_at %s, got %s", fixedNow, loaded.UpdatedAt)
	}
	if len(loaded.RecentRuns) != 1 {
		t.Fatalf("expected 1 recent run, got %d", len(loaded.RecentRuns))
	}
	if !loaded.RecentRuns[0].StartedAt.Equal(state.RecentRuns[0].StartedAt.UTC()) {
		t.Fatalf("expected run timestamp to be normalized, got %#v", loaded.RecentRuns[0])
	}
}

func TestStateStoreRecordRunPrunesRecentRuns(t *testing.T) {
	store := NewStateStore(filepath.Join(t.TempDir(), stateFileName))
	store.MaxRecentRuns = 1

	first := time.Date(2026, time.March, 23, 10, 0, 0, 0, time.UTC)
	second := first.Add(2 * time.Hour)
	current := first
	store.Now = func() time.Time { return current }

	if err := store.RecordRun(WrapperRun{
		Flow:    "ci",
		Lane:    "preview",
		Project: "demo",
		Status:  "ok",
	}); err != nil {
		t.Fatalf("unexpected first record error: %v", err)
	}

	current = second
	if err := store.RecordRun(WrapperRun{
		Flow:    "release",
		Lane:    "production",
		Project: "demo",
		Status:  "failed",
		Notes:   "publish step failed",
	}); err != nil {
		t.Fatalf("unexpected second record error: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if loaded.SelectedFlow != "release" ||
		loaded.SelectedLane != "production" ||
		loaded.ProjectName != "demo" {
		t.Fatalf("expected last run selection to persist, got %#v", loaded)
	}
	if !loaded.LastRunAt.Equal(second.UTC()) {
		t.Fatalf("expected last run timestamp %s, got %s", second.UTC(), loaded.LastRunAt)
	}
	if len(loaded.RecentRuns) != 1 {
		t.Fatalf("expected pruned recent run list, got %#v", loaded.RecentRuns)
	}
	if loaded.RecentRuns[0].Flow != "release" || loaded.RecentRuns[0].Lane != "production" {
		t.Fatalf("expected most recent run to stay at the front, got %#v", loaded.RecentRuns[0])
	}
}

func TestStateStoreLoadMissingFileReturnsDefaultState(t *testing.T) {
	store := NewStateStore(filepath.Join(t.TempDir(), "missing", stateFileName))

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if loaded.Version != stateSchemaV1 {
		t.Fatalf("expected default version %d, got %d", stateSchemaV1, loaded.Version)
	}
	if loaded.SelectedFlow != "" || loaded.SelectedLane != "" || loaded.ProjectName != "" {
		t.Fatalf("expected empty default state, got %#v", loaded)
	}
}
