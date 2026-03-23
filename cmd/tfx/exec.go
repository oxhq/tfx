package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type commandLogWriter struct {
	mu        sync.Mutex
	prefix    string
	onLine    func(string)
	remainder string
}

func (w *commandLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.remainder += string(p)
	for {
		index := strings.IndexByte(w.remainder, '\n')
		if index < 0 {
			break
		}
		line := strings.TrimSpace(strings.TrimSuffix(w.remainder[:index], "\r"))
		if line != "" && w.onLine != nil {
			w.onLine(fmt.Sprintf("%s %s", w.prefix, line))
		}
		w.remainder = w.remainder[index+1:]
	}

	return len(p), nil
}

func (w *commandLogWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	line := strings.TrimSpace(strings.TrimSuffix(w.remainder, "\r"))
	if line != "" && w.onLine != nil {
		w.onLine(fmt.Sprintf("%s %s", w.prefix, line))
	}
	w.remainder = ""
}

func (a *app) runConfiguredCommand(
	ctx context.Context,
	step wrapperStep,
	projectName string,
	releaseLane string,
	secretEnv string,
	secretValue string,
) (ArtifactSummary, error) {
	if len(step.Command) == 0 {
		return ArtifactSummary{}, fmt.Errorf("step %q has empty command", step.ID)
	}

	before, err := discoverConventionalArtifacts(step.Dir)
	if err != nil {
		return ArtifactSummary{}, fmt.Errorf("inspect artifacts before %s: %w", step.ID, err)
	}

	// #nosec G204 -- command argv comes from the explicit project config and is executed without shell interpolation.
	cmd := exec.CommandContext(ctx, step.Command[0], step.Command[1:]...)
	cmd.Dir = step.Dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TFX_PROJECT=%s", projectName),
		fmt.Sprintf("TFX_LANE=%s", releaseLane),
	)
	if secretEnv != "" && secretValue != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", secretEnv, secretValue))
	}
	for key, value := range step.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdout := &commandLogWriter{prefix: "[" + step.ID + "]", onLine: a.appendLog}
	stderr := &commandLogWriter{prefix: "[" + step.ID + " !]", onLine: a.appendLog}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	stdout.flush()
	stderr.flush()

	artifactSummary, artifactErr := buildStepArtifactSummary(step, before)
	switch {
	case runErr != nil:
		return artifactSummary, fmt.Errorf("%s: %w", step.ID, runErr)
	case artifactErr != nil:
		return artifactSummary, fmt.Errorf("%s: %w", step.ID, artifactErr)
	default:
		return artifactSummary, nil
	}
}

func buildStepArtifactSummary(step wrapperStep, before []string) (ArtifactSummary, error) {
	after, err := discoverConventionalArtifacts(step.Dir)
	if err != nil {
		return ArtifactSummary{}, fmt.Errorf("inspect artifacts after %s: %w", step.ID, err)
	}
	discovered := diffArtifactPaths(before, after)
	if len(step.Artifacts) == 0 && len(discovered) == 0 {
		return ArtifactSummary{
			BaseDir:   step.Dir,
			StepID:    step.ID,
			StepTitle: step.Title,
		}, nil
	}

	summary, err := BuildArtifactSummary(step.Dir, step.ID, step.Title, step.Artifacts, discovered)
	if err != nil {
		return ArtifactSummary{}, err
	}
	if err := validateRequiredArtifacts(summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func validateRequiredArtifacts(summary ArtifactSummary) error {
	for _, record := range summary.Records {
		if !record.Declared || record.Optional {
			continue
		}
		switch {
		case record.Problem != "":
			return fmt.Errorf("artifact %s has a problem: %s", record.PathLabel(), record.Problem)
		case !record.Present:
			return fmt.Errorf("required artifact %s is missing", record.PathLabel())
		}
	}
	return nil
}

func discoverConventionalArtifacts(baseDir string) ([]string, error) {
	entries := []string{
		filepath.Join(baseDir, "artifacts"),
		filepath.Join(baseDir, "build"),
		filepath.Join(baseDir, "coverage"),
		filepath.Join(baseDir, "dist"),
	}

	seen := map[string]struct{}{}
	paths := make([]string, 0, 8)
	for _, root := range entries {
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					return nil
				}
				if _, ok := seen[path]; ok {
					return nil
				}
				seen[path] = struct{}{}
				paths = append(paths, path)
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		paths = append(paths, root)
	}

	sort.Strings(paths)
	return paths, nil
}

func diffArtifactPaths(before, after []string) []string {
	seenBefore := make(map[string]struct{}, len(before))
	for _, path := range before {
		seenBefore[path] = struct{}{}
	}

	diff := make([]string, 0, len(after))
	for _, path := range after {
		if _, ok := seenBefore[path]; ok {
			continue
		}
		diff = append(diff, path)
	}
	sort.Strings(diff)
	return diff
}
