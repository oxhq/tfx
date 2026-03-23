package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/progrefx"
	"github.com/oxhq/tfx/runfx"
)

func TestLoadWrapperConfigFromReadsYAMLAndSanitizesPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "tfx.yaml")
	data := []byte(`
project: sample
description: Sample pipeline
lanes: [preview, production]
default_lane: production
secret_env: APP_SECRET
working_dir: workspace
steps:
  - id: build
    title: Build
    command: [go, build, ./...]
    dir: app
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, cwd, resolvedPath, err := loadWrapperConfigFrom(root)
	if err != nil {
		t.Fatalf("unexpected config load error: %v", err)
	}

	if cwd != root {
		t.Fatalf("expected cwd %q, got %q", root, cwd)
	}
	if resolvedPath != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, resolvedPath)
	}
	if cfg.SecretPrompt != "Secret value" {
		t.Fatalf("expected default secret prompt, got %q", cfg.SecretPrompt)
	}
	if cfg.WorkingDir != filepath.Join(root, "workspace") {
		t.Fatalf("expected working dir to be sanitized, got %q", cfg.WorkingDir)
	}
	if got := cfg.Steps[0].Dir; got != filepath.Join(root, "workspace", "app") {
		t.Fatalf("expected step dir to be sanitized, got %q", got)
	}
	if cfg.DefaultLane != "production" {
		t.Fatalf("expected default lane to persist, got %q", cfg.DefaultLane)
	}
}

func TestLoadWrapperConfigFromFallsBackToNearestGoModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := []byte("module example.com/sample\n\ngo 1.24.0\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), goMod, 0o600); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	nested := filepath.Join(root, "cmd", "nested")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	cfg, cwd, configPath, err := loadWrapperConfigFrom(nested)
	if err != nil {
		t.Fatalf("unexpected fallback load error: %v", err)
	}

	if cwd != nested {
		t.Fatalf("expected cwd %q, got %q", nested, cwd)
	}
	if configPath != "" {
		t.Fatalf("expected no explicit config path, got %q", configPath)
	}
	if cfg.WorkingDir != root {
		t.Fatalf("expected fallback working dir %q, got %q", root, cfg.WorkingDir)
	}
	if len(cfg.Steps) != 3 {
		t.Fatalf("expected fallback go pipeline, got %d steps", len(cfg.Steps))
	}
	if got := strings.Join(cfg.Steps[0].Command, " "); got != "go test ./..." {
		t.Fatalf("expected go test step, got %q", got)
	}
}

func TestLoadWrapperConfigFromSupportsNamedFlowsAndProfilesAlias(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "tfx.yaml")
	data := []byte(`
project: sample
description: Sample pipeline
lanes: [preview, production]
default_lane: preview
working_dir: workspace
default_profile: release
steps:
  - id: shared
    title: Shared
    command: [go, test, ./...]
profiles:
  - name: ci
    description: CI checks
  - name: release
    description: Release flow
    lanes: [production]
    require_approval: true
    secret_env: RELEASE_TOKEN
    steps:
      - id: publish
        title: Publish
        command: [go, build, ./...]
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "workspace"), 0o750); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	cfg, _, _, err := loadWrapperConfigFrom(root)
	if err != nil {
		t.Fatalf("unexpected config load error: %v", err)
	}

	if got := cfg.defaultPlanName(); got != "release" {
		t.Fatalf("expected default named flow to be release, got %q", got)
	}
	if names := cfg.namedPlanNames(); len(names) != 2 || names[0] != "ci" || names[1] != "release" {
		t.Fatalf("unexpected named flow list: %#v", names)
	}

	ciPlan := cfg.planFor("ci")
	if len(ciPlan.Steps) != 1 || ciPlan.Steps[0].ID != "shared" {
		t.Fatalf("expected ci flow to inherit shared step, got %#v", ciPlan.Steps)
	}

	releasePlan := cfg.planFor("release")
	if !releasePlan.RequireApproval {
		t.Fatal("expected release flow to require approval")
	}
	if releasePlan.SecretEnv != "RELEASE_TOKEN" {
		t.Fatalf("expected release flow secret env to be overridden, got %q", releasePlan.SecretEnv)
	}
	if releasePlan.DefaultLane != "production" {
		t.Fatalf(
			"expected release flow default lane to be production, got %q",
			releasePlan.DefaultLane,
		)
	}
	if len(releasePlan.Steps) != 1 || releasePlan.Steps[0].ID != "publish" {
		t.Fatalf("expected release flow to use publish step, got %#v", releasePlan.Steps)
	}
}

func TestRunConfiguredCommandUsesEnvAndWorkingDir(t *testing.T) {
	t.Parallel()

	workingDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workingDir, 0o750); err != nil {
		t.Fatalf("failed to create working dir: %v", err)
	}

	app := newTestApp(t, wrapperConfig{
		Project:     "demo",
		Lanes:       []string{"preview"},
		DefaultLane: "preview",
		SecretEnv:   "APP_SECRET",
		WorkingDir:  workingDir,
		Steps: []wrapperStep{
			{
				ID:      "env",
				Title:   "Env",
				Command: helperCommand(t, "success"),
				Env:     map[string]string{"STEP_NAME": "env"},
			},
		},
	})

	summary, err := app.runConfiguredCommand(context.Background(), wrapperStep{
		ID:      "env",
		Title:   "Env",
		Command: helperCommand(t, "success"),
		Dir:     workingDir,
		Env:     map[string]string{"STEP_NAME": "env"},
	}, "demo", "preview", "APP_SECRET", "token")
	if err != nil {
		t.Fatalf("unexpected configured command error: %v", err)
	}
	if summary.Count() != 0 {
		t.Fatalf("expected no artifacts from env helper, got %#v", summary)
	}

	resolvedWorkingDir := workingDir
	if path, err := filepath.EvalSymlinks(workingDir); err == nil {
		resolvedWorkingDir = path
	}

	logs := strings.Join(app.logs, "\n")
	for _, want := range []string{
		"[env] helper success",
		"[env] project=demo",
		"[env] lane=preview",
		"[env] secret=token",
		"[env] step=env",
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected logs to contain %q, got %q", want, logs)
		}
	}
	if !strings.Contains(logs, "[env] cwd="+workingDir) &&
		!strings.Contains(logs, "[env] cwd="+resolvedWorkingDir) {
		t.Fatalf(
			"expected logs to contain working dir %q or %q, got %q",
			workingDir,
			resolvedWorkingDir,
			logs,
		)
	}
}

func TestAppConfiguredFormUsesDefaultsAndSkipsOptionalStages(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:     "demo",
		Lanes:       []string{"preview", "canary"},
		DefaultLane: "preview",
		WorkingDir:  t.TempDir(),
		Steps: []wrapperStep{
			{ID: "noop", Title: "Noop", Command: helperCommand(t, "success")},
		},
	})

	app.activeTab = tabForms
	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})
	if app.formStage != formStageLane || app.projectName != "demo" {
		t.Fatalf(
			"expected default project to advance to lane stage, got stage=%v project=%q",
			app.formStage,
			app.projectName,
		)
	}

	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})
	if app.formStage != formStageDone || app.activeTab != tabFlow {
		t.Fatalf(
			"expected optional stages to be skipped, got stage=%v tab=%v",
			app.formStage,
			app.activeTab,
		)
	}
	if app.releaseLane != "preview" {
		t.Fatalf("expected default lane preview, got %q", app.releaseLane)
	}

	snapshot := string(app.RenderSnapshot())
	for _, want := range []string{"Approval: n/a", "Secret: n/a", "Project: demo"} {
		if !strings.Contains(snapshot, want) {
			t.Fatalf("expected snapshot to contain %q, got %q", want, snapshot)
		}
	}
}

func TestAppNamedFlowSelectionSwitchesPlan(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:     "demo",
		Lanes:       []string{"preview", "production"},
		DefaultLane: "preview",
		WorkingDir:  t.TempDir(),
		Flows: []wrapperProfile{
			{
				Name:        "ci",
				Description: "CI checks",
				Steps: []wrapperStep{
					{ID: "test", Title: "Test", Command: helperCommand(t, "success")},
				},
			},
			{
				Name:            "release",
				Description:     "Release flow",
				Lanes:           []string{"canary", "production"},
				DefaultLane:     "canary",
				RequireApproval: boolPtr(true),
				SecretPrompt:    "Release token",
				SecretEnv:       "APP_SECRET",
				Steps: []wrapperStep{
					{ID: "publish", Title: "Publish", Command: helperCommand(t, "success")},
				},
			},
		},
	})

	app.activeTab = tabForms
	typeRunes(app, "demo")
	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})
	if app.formStage != formStageProfile {
		t.Fatalf("expected named-flow selection stage, got %v", app.formStage)
	}

	_ = app.OnKey(runfx.Key{Code: runfx.KeyArrowDown})
	_ = app.OnKey(runfx.Key{Code: runfx.KeyEnter})

	if app.selectedPlan != "release" {
		t.Fatalf("expected selected flow release, got %q", app.selectedPlan)
	}
	if app.formStage != formStageLane {
		t.Fatalf("expected lane stage after selecting flow, got %v", app.formStage)
	}
	if app.releaseLane != "canary" {
		t.Fatalf("expected release flow default lane canary, got %q", app.releaseLane)
	}
	if app.deployReady {
		t.Fatal("expected release flow to require explicit approval")
	}

	steps := app.flowStepper.Steps()
	if len(steps) != 1 || steps[0].Title != "Publish" {
		t.Fatalf("expected flow widgets to switch to release steps, got %#v", steps)
	}

	rendered := string(app.RenderSnapshot())
	for _, want := range []string{"Flow: release", "Description: Release flow", "Release token: missing"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected snapshot to contain %q, got %q", want, rendered)
		}
	}
}

func TestConfiguredFlowContinuesAfterSoftFailure(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, wrapperConfig{
		Project:     "demo",
		Lanes:       []string{"preview"},
		DefaultLane: "preview",
		WorkingDir:  t.TempDir(),
		Steps: []wrapperStep{
			{
				ID:      "prepare",
				Title:   "Prepare",
				Command: helperCommand(t, "success"),
				Env:     map[string]string{"STEP_NAME": "prepare"},
			},
			{
				ID:              "check",
				Title:           "Check",
				Command:         helperCommand(t, "fail"),
				ContinueOnError: true,
			},
			{
				ID:      "finish",
				Title:   "Finish",
				Command: helperCommand(t, "success"),
				Env:     map[string]string{"STEP_NAME": "finish"},
			},
		},
	})

	app.sleep = func(time.Duration) {}
	app.projectName = "demo"
	app.releaseLane = "preview"
	app.formStage = formStageDone
	app.activeTab = tabFlow
	app.syncOverview()

	if stop := app.OnKey(runfx.Key{Code: runfx.KeyEnter}); stop {
		t.Fatal("expected enter to keep app running")
	}

	waitFor(t, time.Second, func() bool {
		app.mu.Lock()
		defer app.mu.Unlock()
		return !app.flowRunning && app.flowDone
	})

	if app.flowError != "" {
		t.Fatalf("expected soft failure to keep flow successful, got error %q", app.flowError)
	}
	if app.flowWarnings != 1 {
		t.Fatalf("expected one flow warning, got %d", app.flowWarnings)
	}

	steps := app.flowStepper.Steps()
	if steps[0].Status != progrefx.StepDone ||
		steps[1].Status != progrefx.StepFailed ||
		steps[2].Status != progrefx.StepDone {
		t.Fatalf("unexpected step statuses after soft failure: %#v", steps)
	}

	waitFor(t, time.Second, func() bool {
		return strings.Contains(
			strings.Join(app.logs, "\n"),
			"configured flow finished with 1 warning(s)",
		)
	})

	logs := strings.Join(app.logs, "\n")
	for _, want := range []string{
		"continuing after check failure",
		"configured flow finished with 1 warning(s)",
		"[finish] helper success",
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected logs to contain %q, got %q", want, logs)
		}
	}
}
