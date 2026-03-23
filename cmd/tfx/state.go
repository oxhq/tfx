package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	stateFileName    = "state.json"
	stateSchemaV1    = 1
	defaultMaxRecent = 10
)

var (
	userConfigDir = os.UserConfigDir
	tempDir       = os.TempDir
)

// WrapperState stores lightweight UI and execution state for cmd/tfx.
type WrapperState struct {
	Version      int          `json:"version"`
	UpdatedAt    time.Time    `json:"updated_at"`
	SelectedFlow string       `json:"selected_flow,omitempty"`
	SelectedLane string       `json:"selected_lane,omitempty"`
	ProjectName  string       `json:"project_name,omitempty"`
	LastRunAt    time.Time    `json:"last_run_at,omitempty"`
	RecentRuns   []WrapperRun `json:"recent_runs,omitempty"`
}

// WrapperRun captures a single wrapper execution.
type WrapperRun struct {
	Flow       string    `json:"flow,omitempty"`
	Lane       string    `json:"lane,omitempty"`
	Project    string    `json:"project,omitempty"`
	Status     string    `json:"status,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	Notes      string    `json:"notes,omitempty"`
}

// StateStore persists WrapperState to disk.
type StateStore struct {
	Path          string
	Now           func() time.Time
	MaxRecentRuns int
}

// DefaultStatePath resolves the deterministic persistence path for cmd/tfx.
func DefaultStatePath() (string, error) {
	if dir, err := userConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "tfx", stateFileName), nil
	}

	if dir := tempDir(); dir != "" {
		return filepath.Join(dir, "tfx", stateFileName), nil
	}

	return "", errors.New("no usable config or temp directory")
}

// DefaultStateStore returns a store using the default persistence path.
func DefaultStateStore() (*StateStore, error) {
	path, err := DefaultStatePath()
	if err != nil {
		return nil, err
	}
	return NewStateStore(path), nil
}

// NewStateStore creates a store for the provided file path.
func NewStateStore(path string) *StateStore {
	return &StateStore{
		Path:          path,
		Now:           time.Now,
		MaxRecentRuns: defaultMaxRecent,
	}
}

// Load reads the wrapper state from disk.
func (s *StateStore) Load() (WrapperState, error) {
	if s == nil {
		return WrapperState{Version: stateSchemaV1}, nil
	}

	path := s.Path
	if path == "" {
		resolved, err := DefaultStatePath()
		if err != nil {
			return WrapperState{}, err
		}
		path = resolved
	}

	// #nosec G304 -- state path is resolved from the user's config directory or an injected test path.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return WrapperState{Version: stateSchemaV1}, nil
		}
		return WrapperState{}, fmt.Errorf("read wrapper state: %w", err)
	}

	var state WrapperState
	if err := json.Unmarshal(data, &state); err != nil {
		return WrapperState{}, fmt.Errorf("parse wrapper state: %w", err)
	}

	state.normalize()
	return state, nil
}

// Save writes the wrapper state atomically to disk.
func (s *StateStore) Save(state WrapperState) error {
	if s == nil {
		return errors.New("state store is nil")
	}

	path := s.Path
	if path == "" {
		resolved, err := DefaultStatePath()
		if err != nil {
			return err
		}
		path = resolved
	}

	state.normalize()
	state.UpdatedAt = s.now().UTC()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tfx-state-*.json")
	if err != nil {
		return fmt.Errorf("create temp state file: %w", err)
	}
	tmpName := tmp.Name()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("encode wrapper state: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("set wrapper state permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp wrapper state: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if retryErr := os.Rename(tmpName, path); retryErr != nil {
			_ = os.Remove(tmpName)
			return fmt.Errorf("replace wrapper state: %w", retryErr)
		}
	}

	return nil
}

// TouchSelection updates the current selection and persists it.
func (s *StateStore) TouchSelection(flow, lane, project string) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	state.SelectedFlow = flow
	state.SelectedLane = lane
	state.ProjectName = project

	return s.Save(state)
}

// RecordRun appends a recent run and persists the new state.
func (s *StateStore) RecordRun(run WrapperRun) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	now := s.now().UTC()
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	} else {
		run.StartedAt = run.StartedAt.UTC()
	}
	if run.FinishedAt.IsZero() {
		run.FinishedAt = run.StartedAt
	} else {
		run.FinishedAt = run.FinishedAt.UTC()
	}

	state.SelectedFlow = run.Flow
	state.SelectedLane = run.Lane
	state.ProjectName = run.Project
	state.LastRunAt = run.FinishedAt
	state.RecentRuns = prependRecentRun(state.RecentRuns, run, s.maxRecentRuns())

	return s.Save(state)
}

func (s *StateStore) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *StateStore) maxRecentRuns() int {
	if s != nil && s.MaxRecentRuns > 0 {
		return s.MaxRecentRuns
	}
	return defaultMaxRecent
}

func (state *WrapperState) normalize() {
	if state.Version == 0 {
		state.Version = stateSchemaV1
	}
	state.LastRunAt = state.LastRunAt.UTC()
	if len(state.RecentRuns) == 0 {
		state.RecentRuns = nil
		return
	}
	for i := range state.RecentRuns {
		state.RecentRuns[i].StartedAt = state.RecentRuns[i].StartedAt.UTC()
		state.RecentRuns[i].FinishedAt = state.RecentRuns[i].FinishedAt.UTC()
	}
}

func prependRecentRun(existing []WrapperRun, run WrapperRun, limit int) []WrapperRun {
	if limit <= 0 {
		limit = defaultMaxRecent
	}

	cloned := make([]WrapperRun, 0, min(limit, len(existing)+1))
	cloned = append(cloned, run)
	if len(cloned) >= limit {
		return cloned
	}
	for _, item := range existing {
		if sameRun(item, run) {
			continue
		}
		cloned = append(cloned, item)
		if len(cloned) >= limit {
			break
		}
	}

	return cloned
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sameRun(a, b WrapperRun) bool {
	return a.Flow == b.Flow &&
		a.Lane == b.Lane &&
		a.Project == b.Project &&
		a.Status == b.Status &&
		a.StartedAt.Equal(b.StartedAt) &&
		a.FinishedAt.Equal(b.FinishedAt) &&
		a.Notes == b.Notes
}
