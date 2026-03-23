package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/progrefx"
	"github.com/oxhq/tfx/runfx"
)

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func nonTTYInfo() func() runfx.TTYInfo {
	return func() runfx.TTYInfo {
		return runfx.TTYInfo{IsTTY: false}
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatal("condition was not met before timeout")
}

func typeRunes(target interface{ OnKey(runfx.Key) bool }, value string) {
	for _, r := range value {
		key := runfx.Key{Rune: r}
		if r == ' ' {
			key.Code = runfx.KeySpace
		}
		_ = target.OnKey(key)
	}
}

func helperCommand(t *testing.T, mode string) []string {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve test executable: %v", err)
	}

	return []string{exe, "-test.run=TestTFXHelperProcess", "--", mode}
}

func boolPtr(value bool) *bool {
	return &value
}

func newTestApp(t *testing.T, cfg wrapperConfig) *app {
	t.Helper()

	cwd := cfg.WorkingDir
	if cwd == "" {
		cwd = t.TempDir()
		cfg.WorkingDir = cwd
	}

	store := NewStateStore(filepath.Join(t.TempDir(), stateFileName))
	model, err := newAppWithConfigAndStateStore(&bytes.Buffer{}, nonTTYInfo(), cfg, cwd, "", store)
	if err != nil {
		t.Fatalf("unexpected app creation error: %v", err)
	}

	return model
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %q: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	})
}

func TestMainWritesSnapshotInNonTTYMode(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	oldStdout := os.Stdout
	oldStdin := os.Stdin

	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	inReader, inWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	defer func() {
		os.Stdout = oldStdout
		os.Stdin = oldStdin
		_ = outReader.Close()
		_ = outWriter.Close()
		_ = inReader.Close()
		_ = inWriter.Close()
	}()

	os.Stdout = outWriter
	os.Stdin = inReader

	main()
	_ = outWriter.Close()

	data, err := io.ReadAll(outReader)
	if err != nil {
		t.Fatalf("failed to read main output: %v", err)
	}
	output := string(data)
	for _, want := range []string{"TFX wrapper snapshot", "Overview", "Flow", "Forms", "Logs"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected snapshot to contain %q, got %q", want, output)
		}
	}
}

func TestRunFallsBackToSnapshotWithoutTTY(t *testing.T) {
	t.Parallel()

	withWorkingDir(t, t.TempDir())

	var out bytes.Buffer
	if err := run(&out, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"TFX wrapper snapshot",
		"Configured runner state",
		"Press enter or s to run the configured command pipeline",
		"Current prompt: project name",
		"wrapper initialized",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected snapshot to contain %q, got %q", want, rendered)
		}
	}
}

func TestRunWithArgsEnvVersionOnly(t *testing.T) {
	t.Parallel()

	originalVersion := Version
	originalCommit := Commit
	originalBuildDate := BuildDate
	Version = "v0.1.0"
	Commit = "abc1234"
	BuildDate = "2026-03-23T21:00:00Z"
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		BuildDate = originalBuildDate
	})

	var out bytes.Buffer
	if err := runWithArgsEnv(&out, bytes.NewBuffer(nil), []string{"--version"}, nil); err != nil {
		t.Fatalf("unexpected version error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{"tfx v0.1.0", "commit=abc1234", "built=2026-03-23T21:00:00Z"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected version output to contain %q, got %q", want, rendered)
		}
	}
}

func TestParseRunOptionsPrefersFlagOverEnv(t *testing.T) {
	t.Parallel()

	options, err := parseRunOptions(
		[]string{"--flow", "release", "--lane", "production", "--run"},
		func(key string) (string, bool) {
			switch key {
			case "TFX_FLOW":
				return "ci", true
			case "TFX_LANE":
				return "preview", true
			case "TFX_RUN":
				return "false", true
			}
			return "", false
		},
	)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if options.flow != "release" {
		t.Fatalf("expected flow override from flag, got %q", options.flow)
	}
	if options.lane != "production" {
		t.Fatalf("expected lane override from flag, got %q", options.lane)
	}
	if !options.run {
		t.Fatal("expected --run to override env and enable direct execution")
	}
}

func TestRunWithArgsEnvPreselectsFlowInSnapshot(t *testing.T) {
	root := t.TempDir()
	config := []byte(`
project: demo
working_dir: .
default_flow: ci
flows:
  - name: ci
    description: CI checks
    steps:
      - id: test
        title: Test
        command: [go, test, ./...]
  - name: release
    description: Release flow
    default_lane: production
    require_approval: true
    secret_prompt: Release token
    steps:
      - id: publish
        title: Publish
        command: [go, build, ./...]
`)
	// #nosec G304,G703 -- test config is written into a temp directory controlled by the test.
	if err := os.WriteFile(filepath.Join(root, "tfx.yaml"), config, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	withWorkingDir(t, root)

	var out bytes.Buffer
	err := runWithArgsEnv(
		&out,
		bytes.NewBuffer(nil),
		[]string{"--flow=release", "--lane=production"},
		func(string) (string, bool) { return "", false },
	)
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"Flow: release",
		"Selected flow: release",
		"Lane: production",
		"Release token: missing",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected snapshot to contain %q, got %q", want, rendered)
		}
	}
}

func TestRunWithArgsEnvRunsFlowDirectlyInSnapshotMode(t *testing.T) {
	root := t.TempDir()
	originalConfigDir := userConfigDir
	originalTempDir := tempDir
	t.Cleanup(func() {
		userConfigDir = originalConfigDir
		tempDir = originalTempDir
	})
	userConfigDir = func() (string, error) { return filepath.Join(root, "state"), nil }
	tempDir = func() string { return root }

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve test executable: %v", err)
	}
	config := []byte(fmt.Sprintf(`
project: demo
working_dir: .
flows:
  - name: ci
    description: CI checks
    secret_env: APP_SECRET
    steps:
      - id: publish
        title: Publish
        command: [%q, "-test.run=TestTFXHelperProcess", "--", "success"]
`, exe))
	// #nosec G304,G703 -- test config is written into a temp directory controlled by the test.
	if err := os.WriteFile(filepath.Join(root, "tfx.yaml"), config, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	withWorkingDir(t, root)

	var out bytes.Buffer
	err = runWithArgsEnv(
		&out,
		bytes.NewBuffer(nil),
		[]string{"--run"},
		func(key string) (string, bool) {
			if key == "APP_SECRET" {
				return "token", true
			}
			return "", false
		},
	)
	if err != nil {
		t.Fatalf("unexpected direct-run error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		"Configured flow complete",
		"[publish] helper success",
		"Status: tab=Flow",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected direct-run snapshot to contain %q, got %q", want, rendered)
		}
	}
}

func TestRunWithArgsEnvRunRejectsUnknownLane(t *testing.T) {
	root := t.TempDir()
	config := []byte(`
project: demo
working_dir: .
flows:
  - name: release
    lanes: [canary, production]
    steps:
      - id: publish
        title: Publish
        command: [go, build, ./...]
`)
	// #nosec G304,G703 -- test config is written into a temp directory controlled by the test.
	if err := os.WriteFile(filepath.Join(root, "tfx.yaml"), config, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	withWorkingDir(t, root)

	err := runWithArgsEnv(
		&bytes.Buffer{},
		bytes.NewBuffer(nil),
		[]string{"--flow=release", "--lane=preview", "--run"},
		func(string) (string, bool) { return "", false },
	)
	if err == nil || !strings.Contains(err.Error(), `lane "preview" is not defined`) {
		t.Fatalf("expected invalid lane error, got %v", err)
	}
}

func TestRunWithArgsEnvRejectsUnknownFlow(t *testing.T) {
	root := t.TempDir()
	config := []byte(`
project: demo
flows:
  - name: ci
    steps:
      - id: test
        title: Test
        command: [go, test, ./...]
`)
	// #nosec G304,G703 -- test config is written into a temp directory controlled by the test.
	if err := os.WriteFile(filepath.Join(root, "tfx.yaml"), config, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	withWorkingDir(t, root)

	err := runWithArgsEnv(
		&bytes.Buffer{},
		bytes.NewBuffer(nil),
		nil,
		func(key string) (string, bool) {
			if key == "TFX_FLOW" {
				return "release", true
			}
			return "", false
		},
	)
	if err == nil || !strings.Contains(err.Error(), `flow "release" is not defined`) {
		t.Fatalf("expected unknown flow error, got %v", err)
	}
}

func TestAppTabCyclingAndQuit(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:     "demo",
		Lanes:       []string{"preview", "production"},
		DefaultLane: "preview",
		WorkingDir:  t.TempDir(),
		Steps: []wrapperStep{
			{ID: "noop", Title: "Noop", Command: helperCommand(t, "success")},
		},
	})

	app.OnResize(120, 40)
	if app.cols != 120 || app.rows != 40 {
		t.Fatalf("expected resize to persist dimensions, got cols=%d rows=%d", app.cols, app.rows)
	}

	app.Tick(time.Now())
	if stop := app.OnKey(runfx.Key{Rune: ']'}); stop {
		t.Fatal("expected next-tab key to keep app running")
	}
	if app.activeTab != tabFlow {
		t.Fatalf("expected flow tab after cycling, got %v", app.activeTab)
	}

	if stop := app.OnKey(runfx.Key{Rune: '['}); stop {
		t.Fatal("expected previous-tab key to keep app running")
	}
	if app.activeTab != tabOverview {
		t.Fatalf("expected overview tab after cycling back, got %v", app.activeTab)
	}

	if stop := app.OnKey(runfx.Key{Code: runfx.KeyEscape}); !stop {
		t.Fatal("expected escape to stop the app")
	}
}

func TestAppFormWorkflowAndSnapshot(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:         "demo",
		Lanes:           []string{"preview", "canary", "production"},
		DefaultLane:     "preview",
		RequireApproval: true,
		SecretPrompt:    "Release token",
		SecretEnv:       "APP_SECRET",
		WorkingDir:      t.TempDir(),
		Steps: []wrapperStep{
			{ID: "noop", Title: "Noop", Command: helperCommand(t, "success")},
		},
	})

	app.activeTab = tabForms

	typeRunes(app, "tfx")
	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})
	if app.formStage != formStageLane {
		t.Fatalf("expected lane stage after project input, got %v", app.formStage)
	}

	_ = app.OnKey(runfx.Key{Code: runfx.KeyArrowDown})
	_ = app.OnKey(runfx.Key{Code: runfx.KeyArrowDown})
	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})
	if app.formStage != formStageApproval || app.releaseLane != "production" {
		t.Fatalf(
			"expected production lane before approval, got stage=%v lane=%q",
			app.formStage,
			app.releaseLane,
		)
	}

	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})
	if app.formStage != formStageSecret || !app.deployReady {
		t.Fatalf(
			"expected approval to move into secret stage, got stage=%v approval=%v",
			app.formStage,
			app.deployReady,
		)
	}

	typeRunes(app, "token")
	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})
	typeRunes(app, "token")
	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})

	if app.formStage != formStageDone {
		t.Fatalf("expected form completion, got stage=%v", app.formStage)
	}
	if app.activeTab != tabFlow {
		t.Fatalf("expected form completion to switch to flow tab, got %v", app.activeTab)
	}
	if app.projectName != "tfx" || app.secretValue != "token" {
		t.Fatalf(
			"unexpected collected form values: project=%q secret=%q",
			app.projectName,
			app.secretValue,
		)
	}

	rendered := string(app.RenderSnapshot())
	for _, want := range []string{"Project: tfx", "Lane: production", "Release token: *****"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected form snapshot to contain %q, got %q", want, rendered)
		}
	}
}

func TestAppFlowLifecycleAndReset(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:         "demo",
		Description:     "Test flow",
		Lanes:           []string{"preview", "production"},
		DefaultLane:     "preview",
		RequireApproval: true,
		SecretPrompt:    "Release token",
		SecretEnv:       "APP_SECRET",
		WorkingDir:      t.TempDir(),
		Steps: []wrapperStep{
			{
				ID:      "prepare",
				Title:   "Prepare",
				Detail:  "Prepare release assets",
				Command: helperCommand(t, "success"),
				Env:     map[string]string{"STEP_NAME": "prepare"},
			},
			{
				ID:      "release",
				Title:   "Release",
				Detail:  "Publish release assets",
				Command: helperCommand(t, "success"),
				Lanes:   []string{"production"},
				Env:     map[string]string{"STEP_NAME": "release"},
			},
		},
	})

	app.sleep = func(time.Duration) {}
	app.projectName = "demo"
	app.releaseLane = "production"
	app.deployReady = true
	app.secretValue = "token"
	app.formStage = formStageDone
	app.syncOverview()

	app.activeTab = tabFlow
	if stop := app.OnKey(runfx.Key{Code: runfx.KeyEnter}); stop {
		t.Fatal("expected enter to start flow without stopping app")
	}

	waitFor(t, time.Second, func() bool {
		app.mu.Lock()
		defer app.mu.Unlock()
		return !app.flowRunning &&
			app.flowDone &&
			app.flowError == "" &&
			strings.Contains(strings.Join(app.logs, "\n"), "configured flow finished successfully")
	})

	steps := app.flowStepper.Steps()
	for i, step := range steps {
		if step.Status != progrefx.StepDone {
			t.Fatalf("expected completed flow step %d, got %v", i, step.Status)
		}
	}
	if !strings.Contains(app.flowProgress.Render(), "100%") {
		t.Fatalf("expected flow progress to finish, got %q", app.flowProgress.Render())
	}
	if got := strings.Join(app.logs, "\n"); !strings.Contains(got, "starting configured flow") ||
		!strings.Contains(got, "configured flow finished successfully") ||
		!strings.Contains(got, "[prepare] helper success") {
		t.Fatalf("expected flow logs to be populated, got %q", got)
	}

	if stop := app.OnKey(runfx.Key{Rune: 'r'}); stop {
		t.Fatal("expected flow reset to keep app running")
	}
	if app.flowDone || app.flowError != "" {
		t.Fatalf(
			"expected flow reset to clear final state, got done=%v err=%q",
			app.flowDone,
			app.flowError,
		)
	}
}

func TestAppRestoresPersistedSelection(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	store := NewStateStore(filepath.Join(t.TempDir(), stateFileName))
	if err := store.Save(WrapperState{
		SelectedFlow: "release",
		SelectedLane: "production",
		ProjectName:  "persisted-project",
	}); err != nil {
		t.Fatalf("failed to seed state store: %v", err)
	}

	app, err := newAppWithConfigAndStateStore(
		&bytes.Buffer{},
		nonTTYInfo(),
		wrapperConfig{
			Project:     "demo",
			Lanes:       []string{"preview", "production"},
			DefaultLane: "preview",
			WorkingDir:  workingDir,
			Flows: []wrapperProfile{
				{
					Name: "ci",
					Steps: []wrapperStep{
						{ID: "test", Title: "Test", Command: helperCommand(t, "success")},
					},
				},
				{
					Name:        "release",
					DefaultLane: "production",
					Steps: []wrapperStep{
						{ID: "publish", Title: "Publish", Command: helperCommand(t, "success")},
					},
				},
			},
		},
		workingDir,
		"",
		store,
	)
	if err != nil {
		t.Fatalf("unexpected app creation error: %v", err)
	}

	if app.selectedPlan != "release" {
		t.Fatalf("expected persisted flow to be restored, got %q", app.selectedPlan)
	}
	if app.releaseLane != "production" {
		t.Fatalf("expected persisted lane to be restored, got %q", app.releaseLane)
	}
	if app.projectName != "" {
		t.Fatalf(
			"expected project to remain unconfirmed until prompt submit, got %q",
			app.projectName,
		)
	}
	if got := string(app.projectPrompt.Value); got != "persisted-project" {
		t.Fatalf("expected project prompt default to be restored, got %q", got)
	}
}

func TestAppFlowRunsHooksAndArtifacts(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	store := NewStateStore(filepath.Join(t.TempDir(), stateFileName))
	app, err := newAppWithConfigAndStateStore(
		&bytes.Buffer{},
		nonTTYInfo(),
		wrapperConfig{
			Project:     "demo",
			Lanes:       []string{"preview"},
			DefaultLane: "preview",
			WorkingDir:  workingDir,
			BeforeAll: []wrapperStep{
				{
					ID:      "prep",
					Title:   "Prepare",
					Command: helperCommand(t, "success"),
					Env:     map[string]string{"STEP_NAME": "prep"},
				},
			},
			AfterAll: []wrapperStep{
				{
					ID:      "notify",
					Title:   "Notify",
					Command: helperCommand(t, "success"),
					Env:     map[string]string{"STEP_NAME": "notify"},
				},
			},
			Steps: []wrapperStep{
				{
					ID:      "package",
					Title:   "Package",
					Command: helperCommand(t, "artifact"),
					Env:     map[string]string{"STEP_NAME": "package"},
					Artifacts: []ArtifactDeclaration{
						{Name: "release", Path: "dist/release.txt"},
					},
				},
			},
		},
		workingDir,
		"",
		store,
	)
	if err != nil {
		t.Fatalf("unexpected app creation error: %v", err)
	}

	app.sleep = func(time.Duration) {}
	app.projectName = "demo"
	app.releaseLane = "preview"
	app.formStage = formStageDone
	app.deployReady = true
	app.startFlow()

	waitFor(t, time.Second, func() bool {
		app.mu.Lock()
		defer app.mu.Unlock()
		return !app.flowRunning && app.flowDone && app.flowError == ""
	})

	if len(app.lastReport.Hooks) != 2 {
		t.Fatalf("expected before/after hooks in last report, got %#v", app.lastReport.Hooks)
	}
	if len(app.lastReport.Artifacts) != 1 {
		t.Fatalf("expected artifact summary in last report, got %#v", app.lastReport.Artifacts)
	}
	if app.lastReport.Artifacts[0].Present != 1 {
		t.Fatalf("expected one present artifact, got %#v", app.lastReport.Artifacts[0])
	}
	if got := strings.Join(app.logs, "\n"); !strings.Contains(got, "[prep] helper success") ||
		!strings.Contains(got, "[notify] helper success") ||
		!strings.Contains(got, "[package] artifact created") {
		t.Fatalf("expected hook and artifact logs, got %q", got)
	}
	if rendered := string(app.RenderSnapshot()); !strings.Contains(rendered, "Artifacts") ||
		!strings.Contains(rendered, "1 artifact(s)") {
		t.Fatalf("expected snapshot to show artifacts, got %q", rendered)
	}

	state, err := store.Load()
	if err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if len(state.RecentRuns) != 1 || state.RecentRuns[0].Status != "success" {
		t.Fatalf("expected successful run to persist, got %#v", state.RecentRuns)
	}
}

func TestAppFlowFailureRunsOnFailureHook(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:     "demo",
		Lanes:       []string{"preview"},
		DefaultLane: "preview",
		WorkingDir:  t.TempDir(),
		OnFailure: []wrapperStep{
			{
				ID:      "cleanup",
				Title:   "Cleanup",
				Command: helperCommand(t, "success"),
				Env:     map[string]string{"STEP_NAME": "cleanup"},
			},
		},
		Steps: []wrapperStep{
			{ID: "release", Title: "Release", Command: helperCommand(t, "fail")},
		},
	})

	app.sleep = func(time.Duration) {}
	app.projectName = "demo"
	app.releaseLane = "preview"
	app.formStage = formStageDone
	app.deployReady = true
	app.startFlow()

	waitFor(t, time.Second, func() bool {
		app.mu.Lock()
		defer app.mu.Unlock()
		return !app.flowRunning && app.flowError != ""
	})

	if len(app.lastReport.Hooks) != 1 || app.lastReport.Hooks[0].Phase != "on_failure" {
		t.Fatalf("expected on_failure hook report, got %#v", app.lastReport.Hooks)
	}
	if got := strings.Join(app.logs, "\n"); !strings.Contains(got, "[cleanup] helper success") {
		t.Fatalf("expected on_failure hook logs, got %q", got)
	}
}

func TestAppLogsClearAndRender(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:     "demo",
		Lanes:       []string{"preview"},
		DefaultLane: "preview",
		WorkingDir:  t.TempDir(),
		Steps: []wrapperStep{
			{ID: "noop", Title: "Noop", Command: helperCommand(t, "success")},
		},
	})

	for i := 0; i < maxLogLines+3; i++ {
		app.appendLog("line")
	}
	if len(app.logs) != maxLogLines {
		t.Fatalf("expected log trimming to keep %d lines, got %d", maxLogLines, len(app.logs))
	}

	app.activeTab = tabLogs
	if !strings.Contains(string(app.Render()), "Press c to clear the log pane.") {
		t.Fatalf("expected logs view render, got %q", string(app.Render()))
	}
	if stop := app.OnKey(runfx.Key{Rune: 'c'}); stop {
		t.Fatal("expected clear-logs key to keep app running")
	}
	if len(app.logs) != 0 {
		t.Fatalf("expected logs to be cleared, got %d", len(app.logs))
	}
}

func TestRunWithArgsEnvOutputsJSON(t *testing.T) {
	restore := restoreStateResolvers()
	defer restore()

	configDir := t.TempDir()
	userConfigDir = func() (string, error) { return configDir, nil }

	root := t.TempDir()
	config := []byte(fmt.Sprintf(`
project: demo
working_dir: .
steps:
  - id: build
    title: Build
    command: [%q, "-test.run=TestTFXHelperProcess", "--", "artifact"]
    env:
      STEP_NAME: build
    artifacts:
      - name: release
        path: dist/release.txt
`, os.Args[0]))
	// #nosec G304,G703 -- test config is written into a temp directory controlled by the test.
	if err := os.WriteFile(filepath.Join(root, "tfx.yaml"), config, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	withWorkingDir(t, root)

	var out bytes.Buffer
	if err := runWithArgsEnv(
		&out,
		bytes.NewBuffer(nil),
		[]string{"--run", "--json"},
		func(string) (string, bool) { return "", false },
	); err != nil {
		t.Fatalf("unexpected json run error: %v", err)
	}

	rendered := out.String()
	for _, want := range []string{
		`"last_run"`,
		`"status": "success"`,
		`"artifacts"`,
		`"release"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected json output to contain %q, got %q", want, rendered)
		}
	}
}

func TestRunWithArgsEnvOutputsQuietSummary(t *testing.T) {
	restore := restoreStateResolvers()
	defer restore()

	configDir := t.TempDir()
	userConfigDir = func() (string, error) { return configDir, nil }

	root := t.TempDir()
	config := []byte(`
project: demo
working_dir: .
steps:
  - id: build
    title: Build
    command: [go, version]
`)
	// #nosec G304,G703 -- test config is written into a temp directory controlled by the test.
	if err := os.WriteFile(filepath.Join(root, "tfx.yaml"), config, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	withWorkingDir(t, root)

	var out bytes.Buffer
	if err := runWithArgsEnv(
		&out,
		bytes.NewBuffer(nil),
		[]string{"--run", "--quiet"},
		func(string) (string, bool) { return "", false },
	); err != nil {
		t.Fatalf("unexpected quiet run error: %v", err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "status=success") ||
		!strings.Contains(rendered, "flow=default") ||
		!strings.Contains(rendered, "project=demo") {
		t.Fatalf("expected quiet summary, got %q", rendered)
	}
}

func TestRunReturnsWriterErrorInSnapshotMode(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("write failed")
	if err := run(failingWriter{err: wantErr}, bytes.NewBuffer(nil)); !errors.Is(err, wantErr) {
		t.Fatalf("expected writer error %v, got %v", wantErr, err)
	}
}

func TestMainExitsWithErrorWhenStdoutIsClosed(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve test executable: %v", err)
	}
	// #nosec G204,G702 -- the test re-executes the current test binary with a fixed helper flag.
	cmd := exec.Command(exe, "-test.run=TestMainHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_TFX_MAIN_HELPER=1")
	cmd.Stdout = io.Discard

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit error, got %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
	if !strings.Contains(stderr.String(), "tfx:") {
		t.Fatalf("expected stderr to include tfx error prefix, got %q", stderr.String())
	}
}

func TestMainHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TFX_MAIN_HELPER") != "1" {
		return
	}

	_ = os.Stdout.Close()
	main()
	os.Exit(0)
}

func TestTFXHelperProcess(t *testing.T) {
	index := -1
	for i, arg := range os.Args {
		if arg == "--" {
			index = i
			break
		}
	}
	if index < 0 || index >= len(os.Args)-1 {
		return
	}

	switch os.Args[index+1] {
	case "success":
		_, _ = io.WriteString(
			os.Stdout,
			"helper success\n",
		)
		_, _ = io.WriteString(os.Stdout, "project="+os.Getenv("TFX_PROJECT")+"\n")
		_, _ = io.WriteString(os.Stdout, "lane="+os.Getenv("TFX_LANE")+"\n")
		_, _ = io.WriteString(os.Stdout, "secret="+os.Getenv("APP_SECRET")+"\n")
		_, _ = io.WriteString(os.Stdout, "step="+os.Getenv("STEP_NAME")+"\n")
		cwd, err := os.Getwd()
		if err == nil {
			_, _ = io.WriteString(os.Stdout, "cwd="+cwd+"\n")
		}
		os.Exit(0)
	case "artifact":
		if err := os.MkdirAll("dist", 0o750); err != nil {
			_, _ = io.WriteString(os.Stderr, err.Error()+"\n")
			os.Exit(13)
		}
		content := []byte("artifact:" + os.Getenv("STEP_NAME"))
		// #nosec G304,G703 -- helper writes a fixed artifact path under its own test working directory.
		if err := os.WriteFile(filepath.Join("dist", "release.txt"), content, 0o600); err != nil {
			_, _ = io.WriteString(os.Stderr, err.Error()+"\n")
			os.Exit(14)
		}
		_, _ = io.WriteString(os.Stdout, "artifact created\n")
		os.Exit(0)
	case "fail":
		_, _ = io.WriteString(os.Stderr, "helper failure\n")
		os.Exit(7)
	default:
		os.Exit(9)
	}
}
