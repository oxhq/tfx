package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/formfx"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/logfx"
	"github.com/oxhq/tfx/progrefx"
	"github.com/oxhq/tfx/runfx"
)

const (
	maxLogLines   = 18
	flowStepDelay = 80 * time.Millisecond
)

type appTab int

const (
	tabOverview appTab = iota
	tabFlow
	tabForms
	tabLogs
)

func (t appTab) String() string {
	switch t {
	case tabFlow:
		return "Flow"
	case tabForms:
		return "Forms"
	case tabLogs:
		return "Logs"
	default:
		return "Overview"
	}
}

type formStage int

const (
	formStageProject formStage = iota
	formStageProfile
	formStageLane
	formStageApproval
	formStageSecret
	formStageDone
)

type lineLogWriter struct {
	mu     sync.Mutex
	carry  string
	onLine func(string)
}

func (w *lineLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.carry += string(p)
	for {
		index := strings.IndexByte(w.carry, '\n')
		if index < 0 {
			break
		}

		line := strings.TrimSpace(strings.TrimSuffix(w.carry[:index], "\r"))
		if line != "" && w.onLine != nil {
			w.onLine(line)
		}
		w.carry = w.carry[index+1:]
	}

	return len(p), nil
}

type app struct {
	mu         sync.Mutex
	output     io.Writer
	detectTTY  func() runfx.TTYInfo
	sleep      func(time.Duration)
	cfg        wrapperConfig
	cwd        string
	configPath string
	stateStore *StateStore
	statePath  string

	logger *logfx.Logger
	logs   []string

	activeTab appTab
	cols      int
	rows      int

	overviewProgress *progrefx.Progress
	overviewSpinner  *progrefx.Spinner
	overviewStepper  *progrefx.StepperView
	overviewTable    *progrefx.TableView

	flowProgress *progrefx.Progress
	flowSpinner  *progrefx.Spinner
	flowStepper  *progrefx.StepperView
	flowTable    *progrefx.TableView
	flowRunning  bool
	flowDone     bool
	flowError    string
	flowWarnings int
	flowRunID    int
	flowCancel   context.CancelFunc

	projectPrompt *formfx.InputPrompt
	profilePrompt *formfx.SelectPrompt
	lanePrompt    *formfx.SelectPrompt
	approvePrompt *formfx.ConfirmPrompt
	secretPrompt  *formfx.SecretPrompt
	formStage     formStage
	persisted     WrapperState
	recentRuns    []WrapperRun
	lastRunAt     time.Time
	lastReport    wrapperRunReport
	projectName   string
	planOverride  string
	selectedPlan  string
	laneOverride  string
	releaseLane   string
	deployReady   bool
	secretValue   string
}

func newApp(output io.Writer, detectTTY func() runfx.TTYInfo) (*app, error) {
	cfg, cwd, configPath, err := loadWrapperConfig()
	if err != nil {
		return nil, err
	}
	return newAppWithConfig(output, detectTTY, cfg, cwd, configPath)
}

func newAppWithConfig(
	output io.Writer,
	detectTTY func() runfx.TTYInfo,
	cfg wrapperConfig,
	cwd string,
	configPath string,
) (*app, error) {
	return newAppWithConfigAndStateStore(output, detectTTY, cfg, cwd, configPath, nil)
}

func newAppWithConfigAndStateStore(
	output io.Writer,
	detectTTY func() runfx.TTYInfo,
	cfg wrapperConfig,
	cwd string,
	configPath string,
	store *StateStore,
) (*app, error) {
	if output == nil {
		output = io.Discard
	}
	if detectTTY == nil {
		detectTTY = runfx.DetectTTY
	}
	baseDir := cwd
	if configPath != "" {
		baseDir = filepath.Dir(configPath)
	}
	if err := cfg.sanitize(baseDir); err != nil {
		return nil, err
	}

	a := &app{
		output:     output,
		detectTTY:  detectTTY,
		sleep:      time.Sleep,
		activeTab:  tabOverview,
		cfg:        cfg,
		cwd:        cwd,
		configPath: configPath,
	}
	initialPlan := cfg.planFor(cfg.defaultPlanName())

	logOpts := logfx.DefaultOptions()
	logOpts.Output = &lineLogWriter{onLine: a.appendLog}
	logOpts.Format = share.FormatBadge
	logOpts.Timestamp = false
	logOpts.DisableColor = true
	logOpts.Level = share.LevelTrace
	logOpts.Theme = color.GitHubTheme
	logOpts.BadgeWidth = 7
	a.logger = logfx.New(logOpts)

	if store == nil {
		defaultStore, err := DefaultStateStore()
		if err == nil {
			store = defaultStore
		}
	}
	a.stateStore = store
	if store != nil {
		a.statePath = store.Path
		if state, err := store.Load(); err != nil {
			a.logger.Warn(fmt.Sprintf("failed to load wrapper state: %v", err))
		} else {
			a.persisted = state
			a.recentRuns = append([]WrapperRun(nil), state.RecentRuns...)
			a.lastRunAt = state.LastRunAt
		}
	}

	a.overviewProgress = progrefx.Start(progrefx.ProgressConfig{
		Total:     100,
		Label:     "Wrapper readiness",
		Width:     24,
		Theme:     progrefx.NordTheme,
		ShowETA:   true,
		Writer:    output,
		DetectTTY: detectTTY,
	})
	a.overviewProgress.Set(0)

	a.overviewSpinner = progrefx.StartSpinner(progrefx.SpinnerConfig{
		Label:     "Collect release inputs",
		Frames:    progrefx.DefaultSpinnerConfig().Frames,
		Theme:     progrefx.NordTheme,
		Writer:    output,
		DetectTTY: detectTTY,
	})

	a.overviewStepper = progrefx.Stepper(progrefx.StepperConfig{
		Label:     "Milestones",
		Steps:     a.defaultOverviewSteps(),
		Theme:     progrefx.NordTheme,
		Writer:    output,
		DetectTTY: detectTTY,
	})

	a.overviewTable = progrefx.Table(progrefx.TableConfig{
		Title: "Subsystems",
		Columns: []progrefx.TableColumn{
			{Title: "Package"},
			{Title: "Role"},
			{Title: "Status"},
		},
		Theme:     progrefx.NordTheme,
		Style:     progrefx.TableStyleUnicode,
		Writer:    output,
		DetectTTY: detectTTY,
	})

	a.flowProgress = progrefx.Start(progrefx.ProgressConfig{
		Total:     100,
		Label:     flowLabel(initialPlan),
		Width:     24,
		Theme:     progrefx.GitHubTheme,
		ShowETA:   true,
		Writer:    output,
		DetectTTY: detectTTY,
	})
	a.flowProgress.Set(0)

	a.flowSpinner = progrefx.StartSpinner(progrefx.SpinnerConfig{
		Label:     "Idle - press enter to run configured flow",
		Frames:    progrefx.DefaultSpinnerConfig().Frames,
		Theme:     progrefx.GitHubTheme,
		Writer:    output,
		DetectTTY: detectTTY,
	})

	a.flowStepper = progrefx.Stepper(progrefx.StepperConfig{
		Label:     pipelineLabel(initialPlan),
		Steps:     stepItemsForPlan(initialPlan),
		Theme:     progrefx.GitHubTheme,
		Writer:    output,
		DetectTTY: detectTTY,
	})

	a.flowTable = progrefx.Table(progrefx.TableConfig{
		Title: "Pipeline detail",
		Columns: []progrefx.TableColumn{
			{Title: "Step"},
			{Title: "Status"},
			{Title: "Detail"},
		},
		Theme:     progrefx.GitHubTheme,
		Style:     progrefx.TableStyleUnicode,
		Writer:    output,
		DetectTTY: detectTTY,
	})

	if err := a.resetForm(); err != nil {
		return nil, err
	}
	a.resetFlow()
	a.syncOverview()

	a.logger.Badge("TFX", "wrapper initialized", color.ModernBlue)
	if configPath != "" {
		a.logger.Info(fmt.Sprintf("loaded config from %s", configPath))
	} else {
		a.logger.Info("no tfx.yaml found; using fallback project pipeline")
	}
	if len(cfg.namedPlanNames()) > 0 {
		a.logger.Info("complete the form, choose a flow, then run the configured flow")
	} else {
		a.logger.Info("complete the form, then run the configured flow")
	}
	if a.statePath != "" {
		a.logger.Info(fmt.Sprintf("persisting wrapper state to %s", a.statePath))
	}

	return a, nil
}

func (a *app) defaultOverviewSteps() []progrefx.StepItem {
	inputDetail := "Project, lane, approval and secret"
	if len(a.cfg.namedPlans()) > 0 {
		inputDetail = "Project, flow, lane, approval and secret"
	}

	return []progrefx.StepItem{
		{Title: "Load project config", Detail: a.configSummary()},
		{Title: "Collect runtime inputs", Detail: inputDetail},
		{Title: "Run configured pipeline", Detail: "Execute configured commands through flowfx"},
		{Title: "Inspect live output", Detail: "Review command output and final status"},
	}
}

func stepItemsForPlan(plan wrapperPlan) []progrefx.StepItem {
	steps := make([]progrefx.StepItem, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		detail := step.Detail
		if detail == "" {
			detail = strings.Join(step.Command, " ")
		}
		steps = append(steps, progrefx.StepItem{
			Title:  step.Title,
			Detail: detail,
		})
	}
	return steps
}

func (a *app) configSummary() string {
	plan := a.currentPlan()
	origin := a.configPath
	if origin == "" {
		origin = "fallback config"
	}
	if plan.Name == "" {
		return fmt.Sprintf("%s | dir=%s | steps=%d", origin, plan.WorkingDir, len(plan.Steps))
	}
	return fmt.Sprintf(
		"%s | flow=%s | dir=%s | steps=%d",
		origin,
		plan.Name,
		plan.WorkingDir,
		len(plan.Steps),
	)
}

func (a *app) needsApproval() bool {
	return a.currentPlan().RequireApproval
}

func (a *app) needsSecret() bool {
	plan := a.currentPlan()
	return plan.SecretPrompt != "" || plan.SecretEnv != ""
}

func (a *app) secretLabel() string {
	plan := a.currentPlan()
	if plan.SecretPrompt != "" {
		return plan.SecretPrompt
	}
	if plan.SecretEnv != "" {
		return "Secret value"
	}
	return "Secret"
}

func (a *app) persistedPlanNameLocked() string {
	if a.planOverride != "" {
		return a.planOverride
	}
	if containsString(a.cfg.namedPlanNames(), a.persisted.SelectedFlow) {
		return a.persisted.SelectedFlow
	}
	return a.cfg.defaultPlanName()
}

func (a *app) persistedProjectNameLocked() string {
	if strings.TrimSpace(a.projectName) != "" {
		return strings.TrimSpace(a.projectName)
	}
	return strings.TrimSpace(a.persisted.ProjectName)
}

func (a *app) persistedLaneForPlanLocked(plan wrapperPlan) string {
	if a.laneOverride != "" && containsString(plan.Lanes, a.laneOverride) {
		return a.laneOverride
	}
	if containsString(plan.Lanes, a.persisted.SelectedLane) {
		return a.persisted.SelectedLane
	}
	return plan.DefaultLane
}

func (a *app) persistSelection() {
	a.mu.Lock()
	store := a.stateStore
	flow := a.selectedPlan
	lane := a.releaseLane
	project := strings.TrimSpace(a.projectName)
	a.persisted.SelectedFlow = flow
	a.persisted.SelectedLane = lane
	a.persisted.ProjectName = project
	a.mu.Unlock()

	if store == nil {
		return
	}
	if err := store.TouchSelection(flow, lane, project); err != nil {
		a.logger.Warn(fmt.Sprintf("failed to persist wrapper selection: %v", err))
	}
}

func (a *app) currentPlan() wrapperPlan {
	a.mu.Lock()
	selected := a.selectedPlan
	a.mu.Unlock()
	return a.cfg.planFor(selected)
}

func (a *app) currentPlanLocked() wrapperPlan {
	return a.cfg.planFor(a.selectedPlan)
}

func (a *app) hasNamedPlans() bool {
	return len(a.cfg.namedPlanNames()) > 0
}

func (a *app) needsPlanSelectionLocked() bool {
	override := a.planOverride
	return len(a.cfg.namedPlanNames()) > 1 && override == ""
}

func planDisplayName(plan wrapperPlan) string {
	if plan.Name != "" {
		return plan.Name
	}
	return "default"
}

func flowLabel(plan wrapperPlan) string {
	if plan.Name == "" {
		return "Configured flow"
	}
	return fmt.Sprintf("Configured flow: %s", plan.Name)
}

func pipelineLabel(plan wrapperPlan) string {
	if plan.Name == "" {
		return "Configured pipeline"
	}
	return fmt.Sprintf("Configured pipeline: %s", plan.Name)
}

func (a *app) nextStageAfterProjectLocked() formStage {
	if a.needsPlanSelectionLocked() {
		return formStageProfile
	}
	return formStageLane
}

func (a *app) nextStageAfterLaneLocked(plan wrapperPlan) formStage {
	if plan.RequireApproval {
		return formStageApproval
	}
	if plan.SecretPrompt != "" || plan.SecretEnv != "" {
		return formStageSecret
	}
	return formStageDone
}

func (a *app) nextStageAfterApprovalLocked(plan wrapperPlan) formStage {
	if plan.SecretPrompt != "" || plan.SecretEnv != "" {
		return formStageSecret
	}
	return formStageDone
}

func (a *app) appendLog(line string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logs = append(a.logs, line)
	if len(a.logs) > maxLogLines {
		a.logs = append([]string(nil), a.logs[len(a.logs)-maxLogLines:]...)
	}
}

func (a *app) clearLogs() {
	a.mu.Lock()
	a.logs = nil
	a.mu.Unlock()
}

func (a *app) buildLanePrompt(plan wrapperPlan) (*formfx.SelectPrompt, error) {
	selectedIndex := 0
	for i, lane := range plan.Lanes {
		if lane == plan.DefaultLane {
			selectedIndex = i
			break
		}
	}
	return formfx.NewSelectPrompt(formfx.SelectConfig{
		Label:         "Release lane",
		Options:       append([]string(nil), plan.Lanes...),
		SelectedIndex: selectedIndex,
		KeyHandler:    formfx.VerticalKeyHandler,
	})
}

func (a *app) buildApprovalPrompt(plan wrapperPlan) (*formfx.ConfirmPrompt, error) {
	label := "Deploy to the selected lane?"
	if plan.Name != "" {
		label = fmt.Sprintf("Run %s on the selected lane?", plan.Name)
	}
	return formfx.NewConfirmPrompt(&formfx.ConfirmConfig{
		Label:        label,
		DefaultValue: true,
		KeyHandler:   formfx.HorizontalKeyHandler,
	})
}

func (a *app) buildSecretPrompt(plan wrapperPlan) (*formfx.SecretPrompt, error) {
	label := plan.SecretPrompt
	if label == "" && plan.SecretEnv != "" {
		label = "Secret value"
	}
	if label == "" {
		label = "Secret"
	}
	return formfx.NewSecretPrompt(formfx.SecretConfig{
		Label:   label,
		Confirm: true,
		Mask:    '*',
	})
}

func (a *app) applySelectedPlanLocked(name string) error {
	plan := a.cfg.planFor(name)

	lanePrompt, err := a.buildLanePrompt(plan)
	if err != nil {
		return err
	}
	approvePrompt, err := a.buildApprovalPrompt(plan)
	if err != nil {
		return err
	}
	secretPrompt, err := a.buildSecretPrompt(plan)
	if err != nil {
		return err
	}

	a.selectedPlan = plan.Name
	a.lanePrompt = lanePrompt
	a.approvePrompt = approvePrompt
	a.secretPrompt = secretPrompt
	a.releaseLane = a.persistedLaneForPlanLocked(plan)
	a.deployReady = !plan.RequireApproval
	a.secretValue = ""
	return nil
}

func (a *app) selectPlanOverride(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if !a.hasNamedPlans() {
		return fmt.Errorf("flow %q requested, but config does not define named flows", name)
	}
	if !containsString(a.cfg.namedPlanNames(), name) {
		return fmt.Errorf("flow %q is not defined in config", name)
	}

	a.mu.Lock()
	a.planOverride = name
	a.mu.Unlock()

	if err := a.resetForm(); err != nil {
		return err
	}
	a.resetFlow()
	a.persistSelection()
	return nil
}

func (a *app) selectLaneOverride(lane string) error {
	lane = strings.TrimSpace(lane)
	if lane == "" {
		return nil
	}

	plan := a.currentPlan()
	if !containsString(plan.Lanes, lane) {
		return fmt.Errorf("lane %q is not defined for flow %q", lane, planDisplayName(plan))
	}

	a.mu.Lock()
	a.laneOverride = lane
	a.mu.Unlock()

	if err := a.resetForm(); err != nil {
		return err
	}
	a.resetFlow()
	a.persistSelection()
	return nil
}

func (a *app) prepareAutoRun(lookupEnv func(string) (string, bool)) error {
	a.mu.Lock()
	plan := a.currentPlanLocked()
	projectName := strings.TrimSpace(a.projectName)
	if projectName == "" {
		projectName = strings.TrimSpace(a.cfg.Project)
	}
	if projectName == "" {
		projectName = "workspace"
	}

	secretValue := strings.TrimSpace(a.secretValue)
	if secretValue == "" && plan.SecretEnv != "" && lookupEnv != nil {
		if value, ok := lookupEnv(plan.SecretEnv); ok {
			secretValue = strings.TrimSpace(value)
		}
	}

	a.projectName = projectName
	a.releaseLane = plan.DefaultLane
	if a.laneOverride != "" {
		a.releaseLane = a.laneOverride
	}
	a.deployReady = true
	a.secretValue = secretValue
	a.formStage = formStageDone
	a.activeTab = tabFlow

	if err := a.validateRunInputsLocked(); err != nil {
		a.mu.Unlock()
		return err
	}
	a.mu.Unlock()

	a.syncOverview()
	a.persistSelection()
	return nil
}

func (a *app) resetForm() error {
	var profilePrompt *formfx.SelectPrompt
	var err error
	profileNames := a.cfg.namedPlanNames()
	if len(profileNames) > 1 {
		defaultIndex := 0
		a.mu.Lock()
		defaultPlan := a.persistedPlanNameLocked()
		a.mu.Unlock()
		for i, name := range profileNames {
			if name == defaultPlan {
				defaultIndex = i
				break
			}
		}
		profilePrompt, err = formfx.NewSelectPrompt(formfx.SelectConfig{
			Label:         "Configured flow",
			Options:       append([]string(nil), profileNames...),
			SelectedIndex: defaultIndex,
			KeyHandler:    formfx.VerticalKeyHandler,
		})
	}
	if err != nil {
		return err
	}

	a.mu.Lock()
	defaultPlan := a.persistedPlanNameLocked()
	defaultProject := a.persistedProjectNameLocked()
	a.projectPrompt = formfx.NewInputPrompt(defaultProject)
	a.profilePrompt = profilePrompt
	a.formStage = formStageProject
	a.projectName = ""
	if err := a.applySelectedPlanLocked(defaultPlan); err != nil {
		a.mu.Unlock()
		return err
	}
	a.mu.Unlock()

	a.syncOverview()
	return nil
}

func (a *app) resetFlow() {
	a.mu.Lock()
	plan := a.currentPlanLocked()
	a.flowRunID++
	if a.flowCancel != nil {
		a.flowCancel()
		a.flowCancel = nil
	}
	a.flowRunning = false
	a.flowDone = false
	a.flowError = ""
	a.flowWarnings = 0
	a.flowSpinner.SetLabel("Idle - press enter to run configured flow")
	a.flowProgress.Set(0)
	a.flowProgress.SetLabel(flowLabel(plan))
	a.flowStepper.SetLabel(pipelineLabel(plan))
	a.flowStepper.SetSteps(stepItemsForPlan(plan))
	a.updateFlowTableLocked()
	a.mu.Unlock()

	a.syncOverview()
}

func (a *app) updateFlowTableLocked() {
	rows := make([][]string, 0, len(a.flowStepper.Steps()))
	for _, step := range a.flowStepper.Steps() {
		rows = append(rows, []string{
			step.Title,
			stepStatusLabel(step.Status),
			step.Detail,
		})
	}
	a.flowTable.SetRows(rows)
}

func stepStatusLabel(status progrefx.StepStatus) string {
	switch status {
	case progrefx.StepDone:
		return "done"
	case progrefx.StepActive:
		return "active"
	case progrefx.StepFailed:
		return "failed"
	case progrefx.StepSkipped:
		return "skipped"
	default:
		return "pending"
	}
}

func (a *app) formReadyLocked() bool {
	plan := a.currentPlanLocked()
	return a.formStage == formStageDone &&
		a.projectName != "" &&
		a.releaseLane != "" &&
		(!plan.RequireApproval || a.deployReady) &&
		(plan.SecretPrompt == "" && plan.SecretEnv == "" || a.secretValue != "")
}

func (a *app) flowStateLocked() string {
	switch {
	case a.flowRunning:
		return "running"
	case a.flowError != "":
		return "failed"
	case a.flowDone && a.flowWarnings > 0:
		return fmt.Sprintf("done with %d warning(s)", a.flowWarnings)
	case a.flowDone:
		return "done"
	default:
		return "idle"
	}
}

func (a *app) syncOverview() {
	a.mu.Lock()
	formReady := a.formReadyLocked()
	plan := a.currentPlanLocked()
	projectName := a.projectName
	releaseLane := a.releaseLane
	deployReady := a.deployReady
	logCount := len(a.logs)
	flowRunning := a.flowRunning
	flowDone := a.flowDone
	flowError := a.flowError
	flowWarnings := a.flowWarnings
	flowState := a.flowStateLocked()
	lastRunAt := a.lastRunAt
	recentCount := len(a.recentRuns)
	statePath := a.statePath
	a.mu.Unlock()

	progressValue := 10
	spinnerLabel := "Review loaded project config"
	steps := a.defaultOverviewSteps()

	switch {
	case flowDone && flowError == "":
		progressValue = 100
		if flowWarnings > 0 {
			spinnerLabel = "Configured wrapper run completed with warnings"
		} else {
			spinnerLabel = "Configured wrapper run completed"
		}
		steps[0].Status = progrefx.StepDone
		steps[1].Status = progrefx.StepDone
		steps[2].Status = progrefx.StepDone
		steps[3].Status = progrefx.StepDone
	case flowError != "":
		progressValue = 72
		spinnerLabel = "Inspect the failed command output"
		steps[0].Status = progrefx.StepDone
		steps[1].Status = progrefx.StepDone
		steps[2].Status = progrefx.StepFailed
		steps[2].Detail = flowError
		steps[3].Status = progrefx.StepActive
	case flowRunning:
		progressValue = 78
		spinnerLabel = "Configured flow running in the wrapper"
		steps[0].Status = progrefx.StepDone
		steps[1].Status = progrefx.StepDone
		steps[2].Status = progrefx.StepActive
	case formReady:
		progressValue = 55
		spinnerLabel = "Runtime inputs ready for execution"
		steps[0].Status = progrefx.StepDone
		steps[1].Status = progrefx.StepDone
		steps[2].Status = progrefx.StepActive
	case projectName != "":
		progressValue = 28
		spinnerLabel = "Complete the remaining runtime inputs"
		steps[0].Status = progrefx.StepDone
		steps[1].Status = progrefx.StepActive
	default:
		steps[0].Status = progrefx.StepDone
		steps[1].Status = progrefx.StepActive
	}

	a.overviewProgress.Set(progressValue)
	a.overviewSpinner.SetLabel(spinnerLabel)
	a.overviewStepper.SetSteps(steps)
	stateStatus := "session only"
	if statePath != "" {
		stateStatus = fmt.Sprintf(
			"%d recent | last=%s",
			recentCount,
			valueOrDash(formatRunTime(lastRunAt)),
		)
	}
	a.overviewTable.SetRows([][]string{
		{"config", "project pipeline", a.configSummary()},
		{"flow", "selected plan", planDisplayName(plan)},
		{"formfx", "runtime intake", formStatusLabel(projectName, releaseLane, formReady)},
		{"flowfx", "configured pipeline", flowState},
		{"logfx", "event stream", fmt.Sprintf("%d lines", logCount)},
		{"progrefx", "widgets", progressStatusLabel(flowRunning, flowDone, flowError)},
		{"deployment", "approval", boolLabel(deployReady)},
		{"state", "session history", stateStatus},
	})
}

func formStatusLabel(projectName, releaseLane string, ready bool) string {
	switch {
	case ready:
		return "ready"
	case projectName == "":
		return "empty"
	case releaseLane == "":
		return "collecting"
	default:
		return "draft"
	}
}

func progressStatusLabel(flowRunning, flowDone bool, flowError string) string {
	switch {
	case flowDone && flowError == "":
		return "synchronized"
	case flowError != "":
		return "attention"
	case flowRunning:
		return "streaming"
	default:
		return "standby"
	}
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func (a *app) nextTab() {
	a.mu.Lock()
	a.activeTab = (a.activeTab + 1) % 4
	a.mu.Unlock()
}

func (a *app) previousTab() {
	a.mu.Lock()
	a.activeTab = (a.activeTab + 3) % 4
	a.mu.Unlock()
}

func (a *app) startFlow() {
	a.mu.Lock()
	if a.flowRunning {
		a.mu.Unlock()
		a.logger.Warn("flow is already running")
		return
	}

	if err := a.validateRunInputsLocked(); err != nil {
		a.flowError = err.Error()
		a.mu.Unlock()
		a.logger.Error(fmt.Sprintf("cannot start configured flow: %v", err))
		a.syncOverview()
		return
	}

	a.flowRunID++
	runID := a.flowRunID
	if a.flowCancel != nil {
		a.flowCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.flowCancel = cancel
	a.flowRunning = true
	a.flowDone = false
	a.flowError = ""
	projectName := a.projectName
	plan := a.currentPlanLocked()
	releaseLane := a.releaseLane
	secretValue := a.secretValue
	steps := cloneSteps(plan.Steps)
	a.flowSpinner.SetLabel("Running configured pipeline")
	a.flowProgress.Set(0)
	a.flowProgress.SetLabel(flowLabel(plan))
	a.flowStepper.SetLabel(pipelineLabel(plan))
	a.flowStepper.SetSteps(stepItemsForPlan(plan))
	a.updateFlowTableLocked()
	a.mu.Unlock()

	a.syncOverview()
	a.logger.Badge("FLOW", "starting configured flow", color.ModernBlue)

	go func() {
		report, err := a.executeConfiguredFlow(
			ctx,
			runID,
			projectName,
			plan,
			releaseLane,
			secretValue,
			steps,
		)
		a.finishFlow(runID, report, err)
	}()
}

func (a *app) executeConfiguredFlow(
	ctx context.Context,
	runID int,
	projectName string,
	plan wrapperPlan,
	releaseLane string,
	secretValue string,
	steps []wrapperStep,
) (wrapperRunReport, error) {
	report := wrapperRunReport{
		Flow:       planDisplayName(plan),
		Lane:       releaseLane,
		Project:    projectName,
		WorkingDir: plan.WorkingDir,
		ConfigPath: a.configPath,
		StatePath:  a.statePath,
		StartedAt:  time.Now().UTC(),
	}

	runErr := a.runConfiguredPhase(
		ctx,
		runID,
		"before_all",
		plan.BeforeAll,
		projectName,
		plan,
		releaseLane,
		secretValue,
		&report,
		false,
	)
	if runErr == nil {
		runErr = a.runConfiguredPhase(
			ctx,
			runID,
			"step",
			steps,
			projectName,
			plan,
			releaseLane,
			secretValue,
			&report,
			true,
		)
	}
	if runErr == nil {
		runErr = a.runConfiguredPhase(
			ctx,
			runID,
			"after_all",
			plan.AfterAll,
			projectName,
			plan,
			releaseLane,
			secretValue,
			&report,
			false,
		)
	}
	if runErr != nil && len(plan.OnFailure) > 0 {
		hookErr := a.runConfiguredPhase(
			ctx,
			runID,
			"on_failure",
			plan.OnFailure,
			projectName,
			plan,
			releaseLane,
			secretValue,
			&report,
			false,
		)
		if hookErr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("on_failure hook error: %w", hookErr))
		}
	}

	report.FinishedAt = time.Now().UTC()
	report.DurationMillis = report.FinishedAt.Sub(report.StartedAt).Milliseconds()
	switch {
	case runErr != nil:
		report.Status = "failed"
		report.Error = runErr.Error()
	case report.Warnings > 0:
		report.Status = "warning"
	default:
		report.Status = "success"
	}

	return report, runErr
}

func (a *app) validateRunInputsLocked() error {
	plan := a.currentPlanLocked()
	switch {
	case strings.TrimSpace(a.projectName) == "":
		return fmt.Errorf("project name is required")
	case strings.TrimSpace(a.releaseLane) == "":
		return fmt.Errorf("release lane is required")
	case !containsString(plan.Lanes, a.releaseLane):
		return fmt.Errorf("release lane %q is not defined in config", a.releaseLane)
	case plan.RequireApproval && !a.deployReady:
		return fmt.Errorf("configured flow requires approval")
	case (plan.SecretPrompt != "" || plan.SecretEnv != "") &&
		strings.TrimSpace(a.secretValue) == "":
		return fmt.Errorf("secret value is required")
	default:
		return nil
	}
}

func stepAllowedForLane(step wrapperStep, releaseLane string) bool {
	if len(step.Lanes) == 0 {
		return true
	}
	return containsString(step.Lanes, releaseLane)
}

func laneSkipDetail(releaseLane string, step wrapperStep) string {
	if len(step.Lanes) == 0 {
		return step.Detail
	}
	return fmt.Sprintf("Skipped for lane %s", releaseLane)
}

func (a *app) runConfiguredPhase(
	ctx context.Context,
	runID int,
	phase string,
	steps []wrapperStep,
	projectName string,
	plan wrapperPlan,
	releaseLane string,
	secretValue string,
	report *wrapperRunReport,
	updateFlowUI bool,
) error {
	for index, configured := range steps {
		if err := ctx.Err(); err != nil {
			if updateFlowUI {
				a.failFlowStep(runID, index, err.Error())
			}
			return err
		}

		detail := configured.Detail
		if detail == "" {
			detail = strings.Join(configured.Command, " ")
		}
		task := wrapperTaskReport{
			Phase:           phase,
			ID:              configured.ID,
			Title:           configured.Title,
			Status:          "pending",
			Detail:          detail,
			Command:         append([]string(nil), configured.Command...),
			ContinueOnError: configured.ContinueOnError,
			StartedAt:       time.Now().UTC(),
		}

		if !stepAllowedForLane(configured, releaseLane) {
			task.Status = "skipped"
			task.Detail = laneSkipDetail(releaseLane, configured)
			task.FinishedAt = time.Now().UTC()
			task.DurationMillis = task.FinishedAt.Sub(task.StartedAt).Milliseconds()
			appendTaskReport(report, task)
			if updateFlowUI {
				a.skipFlowStep(runID, index, task.Detail)
			} else {
				a.logger.Info(fmt.Sprintf(
					"skipping %s hook %s for lane %s",
					phase,
					configured.ID,
					releaseLane,
				))
			}
			continue
		}

		if updateFlowUI {
			a.updateActiveFlowStep(runID, index, detail)
			a.sleep(flowStepDelay)
			if err := ctx.Err(); err != nil {
				task.Status = "failed"
				task.Detail = err.Error()
				task.FinishedAt = time.Now().UTC()
				task.DurationMillis = task.FinishedAt.Sub(task.StartedAt).Milliseconds()
				appendTaskReport(report, task)
				a.failFlowStep(runID, index, task.Detail)
				return err
			}
		}

		a.logger.Badge(
			"RUN",
			fmt.Sprintf("%s/%s -> %s", phase, configured.ID, strings.Join(configured.Command, " ")),
			color.ModernBlue,
		)
		artifactSummary, err := a.runConfiguredCommand(
			ctx,
			configured,
			projectName,
			releaseLane,
			plan.SecretEnv,
			secretValue,
		)
		task.FinishedAt = time.Now().UTC()
		task.DurationMillis = task.FinishedAt.Sub(task.StartedAt).Milliseconds()
		if artifactSummary.Count() > 0 {
			report.Artifacts = append(
				report.Artifacts,
				artifactReportFromSummary(phase, artifactSummary),
			)
		}

		if err != nil {
			task.Detail = err.Error()
			if configured.ContinueOnError {
				task.Status = "warning"
				report.Warnings++
				appendTaskReport(report, task)
				if updateFlowUI {
					a.warnFlowStep(runID, index, task.Detail)
				} else {
					a.bumpFlowWarnings(runID)
				}
				a.logger.Warn(fmt.Sprintf("continuing after %s failure: %v", configured.ID, err))
				continue
			}

			task.Status = "failed"
			appendTaskReport(report, task)
			if updateFlowUI {
				a.failFlowStep(runID, index, task.Detail)
			}
			return err
		}

		task.Status = "done"
		if artifactSummary.Count() > 0 {
			task.Detail = detail + " | " + artifactSummary.Brief()
		}
		appendTaskReport(report, task)
		if updateFlowUI {
			a.completeFlowStep(runID, index, task.Detail)
		}
	}

	return nil
}

func (a *app) updateActiveFlowStep(runID int, index int, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if runID != a.flowRunID {
		return
	}

	a.flowSpinner.SetLabel(detail)
	a.flowStepper.Activate(index)
	a.flowStepper.SetDetail(index, detail)
	a.updateFlowTableLocked()
}

func (a *app) completeFlowStep(runID int, index int, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if runID != a.flowRunID {
		return
	}

	a.flowStepper.Complete(index)
	a.flowStepper.SetDetail(index, detail)
	totalSteps := len(a.flowStepper.Steps())
	if totalSteps < 1 {
		totalSteps = 1
	}
	a.flowProgress.Set(((index + 1) * 100) / totalSteps)
	a.updateFlowTableLocked()
}

func (a *app) failFlowStep(runID int, index int, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if runID != a.flowRunID {
		return
	}

	a.flowStepper.Fail(index, detail)
	a.flowSpinner.SetLabel("Flow failed")
	a.flowError = detail
	a.updateFlowTableLocked()
}

func (a *app) warnFlowStep(runID int, index int, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if runID != a.flowRunID {
		return
	}

	a.flowStepper.Fail(index, detail)
	a.flowSpinner.SetLabel("Continuing after step failure")
	a.flowWarnings++
	a.updateFlowTableLocked()
}

func (a *app) bumpFlowWarnings(runID int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if runID != a.flowRunID {
		return
	}
	a.flowWarnings++
}

func (a *app) skipFlowStep(runID int, index int, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if runID != a.flowRunID {
		return
	}

	a.flowStepper.Skip(index)
	if detail != "" {
		a.flowStepper.SetDetail(index, detail)
	}
	totalSteps := len(a.flowStepper.Steps())
	if totalSteps < 1 {
		totalSteps = 1
	}
	a.flowProgress.Set(((index + 1) * 100) / totalSteps)
	a.updateFlowTableLocked()
}

func appendTaskReport(report *wrapperRunReport, task wrapperTaskReport) {
	if report == nil {
		return
	}
	if task.Phase == "step" {
		report.Steps = append(report.Steps, task)
		return
	}
	report.Hooks = append(report.Hooks, task)
}

func (a *app) recordRun(report wrapperRunReport) {
	run := WrapperRun{
		Flow:       report.Flow,
		Lane:       report.Lane,
		Project:    report.Project,
		Status:     report.Status,
		StartedAt:  report.StartedAt,
		FinishedAt: report.FinishedAt,
	}
	switch {
	case report.Error != "":
		run.Notes = report.Error
	case report.Warnings > 0:
		run.Notes = fmt.Sprintf("%d warning(s)", report.Warnings)
	case len(report.Artifacts) > 0:
		run.Notes = fmt.Sprintf("%d artifact group(s)", len(report.Artifacts))
	}

	a.mu.Lock()
	store := a.stateStore
	limit := defaultMaxRecent
	if store != nil {
		limit = store.maxRecentRuns()
	}
	a.lastRunAt = run.FinishedAt
	a.persisted.LastRunAt = run.FinishedAt
	a.recentRuns = prependRecentRun(a.recentRuns, run, limit)
	a.lastReport.RecentRuns = append([]WrapperRun(nil), a.recentRuns...)
	a.mu.Unlock()

	if store == nil {
		return
	}
	if err := store.RecordRun(run); err != nil {
		a.logger.Warn(fmt.Sprintf("failed to persist wrapper run: %v", err))
	}
}

func (a *app) finishFlow(runID int, report wrapperRunReport, err error) {
	a.mu.Lock()
	if runID != a.flowRunID {
		a.mu.Unlock()
		return
	}
	runStatus := "success"
	a.flowRunning = false
	a.flowCancel = nil
	if err == nil {
		a.flowDone = true
		a.flowError = ""
		if a.flowWarnings > 0 {
			a.flowSpinner.SetLabel("Configured flow complete with warnings")
		} else {
			a.flowSpinner.SetLabel("Configured flow complete")
		}
		a.flowProgress.Finish()
		if a.flowWarnings > 0 {
			runStatus = "warning"
		}
	} else {
		a.flowDone = false
		a.flowError = err.Error()
		a.flowSpinner.SetLabel("Flow failed")
		runStatus = "failed"
	}
	report.Status = runStatus
	report.Error = a.flowError
	report.Warnings = a.flowWarnings
	report.RecentRuns = append([]WrapperRun(nil), a.recentRuns...)
	a.lastReport = cloneRunReport(report)
	a.updateFlowTableLocked()
	a.mu.Unlock()

	a.recordRun(report)

	switch {
	case err != nil:
		a.logger.Error(fmt.Sprintf("configured flow failed: %v", err))
	case a.flowWarnings > 0:
		a.logger.Warn(fmt.Sprintf("configured flow finished with %d warning(s)", a.flowWarnings))
	default:
		a.logger.Badge("READY", "configured flow finished successfully", color.ModernGreen)
	}
	a.syncOverview()
}

func (a *app) OnResize(cols, rows int) {
	a.mu.Lock()
	a.cols = cols
	a.rows = rows
	a.mu.Unlock()
}

func (a *app) Tick(_ time.Time) {
	a.overviewSpinner.Tick()
	a.flowSpinner.Tick()
	a.syncOverview()
}

func (a *app) OnKey(key runfx.Key) bool {
	if key.IsCancel() || key.Code == runfx.KeyQ || key.Rune == 'q' || key.Rune == 'Q' {
		a.mu.Lock()
		cancel := a.flowCancel
		a.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return true
	}

	switch key.Rune {
	case ']':
		a.nextTab()
		return false
	case '[':
		a.previousTab()
		return false
	}

	a.mu.Lock()
	activeTab := a.activeTab
	a.mu.Unlock()

	switch activeTab {
	case tabFlow:
		return a.handleFlowKey(key)
	case tabForms:
		return a.handleFormKey(key)
	case tabLogs:
		return a.handleLogsKey(key)
	default:
		return a.handleOverviewKey(key)
	}
}

func (a *app) handleOverviewKey(key runfx.Key) bool {
	switch {
	case key.Code == runfx.KeyEnter || key.Rune == 's' || key.Rune == 'S':
		a.startFlow()
	case key.Rune == 'r' || key.Rune == 'R':
		_ = a.resetForm()
		a.resetFlow()
		a.logger.Info("wrapper state reset from overview")
	case key.Rune == 'c' || key.Rune == 'C':
		a.clearLogs()
	}
	return false
}

func (a *app) handleFlowKey(key runfx.Key) bool {
	switch {
	case key.Code == runfx.KeyEnter || key.Rune == 's' || key.Rune == 'S':
		a.startFlow()
	case key.Rune == 'r' || key.Rune == 'R':
		a.resetFlow()
		a.logger.Info("flow state reset")
	case key.Rune == 'c' || key.Rune == 'C':
		a.clearLogs()
	}
	return false
}

func (a *app) handleLogsKey(key runfx.Key) bool {
	switch key.Rune {
	case 'c', 'C':
		a.clearLogs()
	case 'r', 'R':
		a.logger.Info("log pane refreshed")
	}
	return false
}

func (a *app) handleFormKey(key runfx.Key) bool {
	if key.Rune == 'r' || key.Rune == 'R' {
		if err := a.resetForm(); err != nil {
			a.logger.Error(fmt.Sprintf("failed to reset form: %v", err))
		} else {
			a.logger.Info("release form reset")
		}
		return false
	}

	a.mu.Lock()
	stage := a.formStage
	projectPrompt := a.projectPrompt
	profilePrompt := a.profilePrompt
	lanePrompt := a.lanePrompt
	approvePrompt := a.approvePrompt
	secretPrompt := a.secretPrompt
	a.mu.Unlock()

	switch stage {
	case formStageProject:
		if !projectPrompt.OnKey(key) {
			return false
		}
		select {
		case value := <-projectPrompt.Done:
			value = strings.TrimSpace(value)
			if value == "" {
				value = strings.TrimSpace(a.cfg.Project)
			}
			if value == "" {
				a.logger.Warn("project name is required")
				if err := a.resetForm(); err != nil {
					a.logger.Error(fmt.Sprintf("failed to rebuild form: %v", err))
				}
				return false
			}
			a.mu.Lock()
			a.projectName = value
			a.formStage = a.nextStageAfterProjectLocked()
			a.mu.Unlock()
			a.syncOverview()
			a.persistSelection()
			a.logger.Badge("FORM", fmt.Sprintf("project set to %s", value), color.ModernBlue)
		default:
		}
	case formStageProfile:
		if !profilePrompt.OnKey(key) {
			return false
		}
		select {
		case index := <-profilePrompt.Done():
			profileNames := a.cfg.namedPlanNames()
			if index < 0 || index >= len(profileNames) {
				a.logger.Error(fmt.Sprintf("selected flow index %d is out of range", index))
				return false
			}
			selected := profileNames[index]
			a.mu.Lock()
			if err := a.applySelectedPlanLocked(selected); err != nil {
				a.mu.Unlock()
				a.logger.Error(fmt.Sprintf("failed to apply selected flow: %v", err))
				return false
			}
			a.formStage = formStageLane
			a.mu.Unlock()
			a.resetFlow()
			a.syncOverview()
			a.persistSelection()
			a.logger.Info(fmt.Sprintf("selected %s flow", selected))
		default:
		}
	case formStageLane:
		if !lanePrompt.OnKey(key) {
			return false
		}
		select {
		case index := <-lanePrompt.Done():
			plan := a.currentPlan()
			if index < 0 || index >= len(plan.Lanes) {
				a.logger.Error(fmt.Sprintf("selected lane index %d is out of range", index))
				return false
			}
			lane := plan.Lanes[index]
			a.mu.Lock()
			a.releaseLane = lane
			a.formStage = a.nextStageAfterLaneLocked(plan)
			if a.formStage == formStageDone {
				a.activeTab = tabFlow
			}
			a.mu.Unlock()
			a.syncOverview()
			a.persistSelection()
			a.logger.Info(fmt.Sprintf("selected %s lane", lane))
			if a.formStage == formStageDone {
				a.logger.Badge("FORM", "runtime inputs completed", color.ModernGreen)
			}
		default:
		}
	case formStageApproval:
		if !approvePrompt.OnKey(key) {
			return false
		}
		select {
		case value := <-approvePrompt.Done():
			plan := a.currentPlan()
			deployReady := value == 0
			a.mu.Lock()
			a.deployReady = deployReady
			a.formStage = a.nextStageAfterApprovalLocked(plan)
			if a.formStage == formStageDone {
				a.activeTab = tabFlow
			}
			a.mu.Unlock()
			a.syncOverview()
			a.persistSelection()
			if deployReady {
				a.logger.Info("deployment approved")
			} else {
				a.logger.Warn("deployment approval rejected; configured flow remains blocked")
			}
			if a.formStage == formStageDone {
				a.logger.Badge("FORM", "runtime inputs completed", color.ModernGreen)
			}
		default:
		}
	case formStageSecret:
		if !secretPrompt.OnKey(key) {
			return false
		}
		select {
		case value := <-secretPrompt.Done():
			value = strings.TrimSpace(value)
			if value == "" {
				a.logger.Warn(fmt.Sprintf("%s is required", strings.ToLower(a.secretLabel())))
				return false
			}
			a.mu.Lock()
			a.secretValue = value
			a.formStage = formStageDone
			a.activeTab = tabFlow
			a.mu.Unlock()
			a.syncOverview()
			a.persistSelection()
			a.logger.Badge("FORM", "release form completed", color.ModernGreen)
		default:
		}
	case formStageDone:
		if key.Code == runfx.KeyEnter {
			a.mu.Lock()
			a.activeTab = tabFlow
			a.mu.Unlock()
		}
	}

	return false
}

func (a *app) Render() []byte {
	a.syncOverview()

	a.mu.Lock()
	activeTab := a.activeTab
	a.mu.Unlock()

	var body string
	switch activeTab {
	case tabFlow:
		body = a.renderFlowView()
	case tabForms:
		body = a.renderFormView()
	case tabLogs:
		body = a.renderLogsView()
	default:
		body = a.renderOverviewView()
	}

	var buf bytes.Buffer
	buf.WriteString(a.renderHeader())
	buf.WriteString("\n\n")
	buf.WriteString(body)
	buf.WriteString("\n\n")
	buf.WriteString(a.renderFooter())
	buf.WriteString("\n")
	return buf.Bytes()
}

func (a *app) RenderSnapshot() []byte {
	a.syncOverview()

	sections := []struct {
		title string
		body  string
	}{
		{title: "Overview", body: a.renderOverviewView()},
		{title: "Flow", body: a.renderFlowView()},
		{title: "Forms", body: a.renderFormView()},
		{title: "Logs", body: a.renderLogsView()},
	}

	var buf bytes.Buffer
	buf.WriteString("TFX wrapper snapshot\n")
	buf.WriteString("Non-interactive mode detected, rendering all integrated sections.\n\n")
	for i, section := range sections {
		buf.WriteString(section.title)
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat("=", len(section.title)))
		buf.WriteString("\n")
		buf.WriteString(section.body)
		if i < len(sections)-1 {
			buf.WriteString("\n\n")
		}
	}
	buf.WriteString("\n\n")
	buf.WriteString(a.renderFooter())
	buf.WriteString("\n")
	return buf.Bytes()
}

func (a *app) renderHeader() string {
	a.mu.Lock()
	activeTab := a.activeTab
	a.mu.Unlock()

	parts := make([]string, 0, 4)
	for _, tab := range []appTab{tabOverview, tabFlow, tabForms, tabLogs} {
		label := tab.String()
		if tab == activeTab {
			label = "[" + strings.ToUpper(label) + "]"
		}
		parts = append(parts, label)
	}

	return strings.Join([]string{
		"TFX wrapper " + Version,
		"Tabs: " + strings.Join(parts, "  "),
		"Keys: q quit | [ prev | ] next | enter action | s start flow | r reset | c clear logs",
	}, "\n")
}

func (a *app) renderFooter() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	plan := a.currentPlanLocked()

	project := "-"
	if a.projectName != "" {
		project = a.projectName
	}
	lane := "-"
	if a.releaseLane != "" {
		lane = a.releaseLane
	}
	token := "missing"
	if plan.SecretPrompt == "" && plan.SecretEnv == "" {
		token = "not required"
	} else if a.secretValue != "" {
		token = fmt.Sprintf("%d chars", len([]rune(a.secretValue)))
	}
	approval := boolLabel(a.deployReady)
	if !plan.RequireApproval {
		approval = "n/a"
	}

	return fmt.Sprintf(
		"Status: tab=%s | project=%s | plan=%s | lane=%s | approval=%s | token=%s | flow=%s | logs=%d | cwd=%s | size=%dx%d",
		a.activeTab.String(),
		project,
		planDisplayName(plan),
		lane,
		approval,
		token,
		a.flowStateLocked(),
		len(a.logs),
		plan.WorkingDir,
		a.cols,
		a.rows,
	)
}

func (a *app) renderOverviewView() string {
	a.mu.Lock()
	statePath := a.statePath
	lastRunAt := a.lastRunAt
	recentRuns := append([]WrapperRun(nil), a.recentRuns...)
	a.mu.Unlock()

	var buf bytes.Buffer
	buf.WriteString("Overview\n")
	buf.WriteString("Configured runner state for progrefx, formfx, flowfx and logfx.\n\n")
	buf.WriteString("Config: ")
	buf.WriteString(a.configSummary())
	if statePath != "" {
		buf.WriteString("\nState path: ")
		buf.WriteString(statePath)
	}
	if !lastRunAt.IsZero() {
		buf.WriteString("\nLast run: ")
		buf.WriteString(formatRunTime(lastRunAt))
	}
	buf.WriteString("\nRecent runs: ")
	buf.WriteString(formatRecentRuns(recentRuns))
	buf.WriteString("\n\n")
	buf.WriteString(a.overviewSpinner.Render())
	buf.WriteString("\n")
	buf.WriteString(a.overviewProgress.Render())
	buf.WriteString("\n\n")
	buf.WriteString(a.overviewStepper.Render())
	buf.WriteString("\n\n")
	buf.WriteString(a.overviewTable.Render())
	return buf.String()
}

func (a *app) renderFlowView() string {
	a.mu.Lock()
	flowError := a.flowError
	flowWarnings := a.flowWarnings
	plan := a.currentPlanLocked()
	lastReport := cloneRunReport(a.lastReport)
	a.mu.Unlock()

	var buf bytes.Buffer
	buf.WriteString("Flow\n")
	buf.WriteString(
		"Press enter or s to run the configured command pipeline. Press r to reset it.\n\n",
	)
	if plan.Name != "" {
		buf.WriteString("Selected flow: ")
		buf.WriteString(plan.Name)
		buf.WriteString("\n")
	}
	buf.WriteString("Working directory: ")
	buf.WriteString(plan.WorkingDir)
	buf.WriteString("\n")
	buf.WriteString("Config: ")
	buf.WriteString(a.configSummary())
	buf.WriteString("\n\n")
	buf.WriteString(a.flowSpinner.Render())
	buf.WriteString("\n")
	buf.WriteString(a.flowProgress.Render())
	if flowError != "" {
		buf.WriteString("\n")
		buf.WriteString("Last error: ")
		buf.WriteString(flowError)
	} else if flowWarnings > 0 {
		buf.WriteString("\n")
		_, _ = fmt.Fprintf(&buf, "Warnings: %d", flowWarnings)
	}
	buf.WriteString("\n\n")
	buf.WriteString(a.flowStepper.Render())
	buf.WriteString("\n\n")
	buf.WriteString(a.flowTable.Render())
	if len(lastReport.Artifacts) > 0 {
		buf.WriteString("\n\nArtifacts\n")
		buf.WriteString("---------\n")
		for _, artifact := range lastReport.Artifacts {
			_, _ = fmt.Fprintf(
				&buf,
				"%s/%s: %s\n",
				valueOrDash(artifact.Phase),
				valueOrDash(artifact.StepID),
				artifact.Brief,
			)
		}
	}
	return buf.String()
}

func (a *app) renderFormView() string {
	a.mu.Lock()
	stage := a.formStage
	projectName := a.projectName
	selectedPlan := a.selectedPlan
	releaseLane := a.releaseLane
	deployReady := a.deployReady
	secretValue := a.secretValue
	projectPrompt := a.projectPrompt
	profilePrompt := a.profilePrompt
	lanePrompt := a.lanePrompt
	approvePrompt := a.approvePrompt
	secretPrompt := a.secretPrompt
	a.mu.Unlock()
	plan := a.currentPlan()

	var buf bytes.Buffer
	buf.WriteString("Forms\n")
	buf.WriteString(
		"Use the live prompts to collect runtime inputs for the configured flow. Press r to restart the form.\n\n",
	)
	_, _ = fmt.Fprintf(&buf, "Project: %s\n", valueOrDash(projectName))
	if a.hasNamedPlans() {
		_, _ = fmt.Fprintf(&buf, "Flow: %s\n", valueOrDash(selectedPlan))
	}
	_, _ = fmt.Fprintf(&buf, "Lane: %s\n", valueOrDash(releaseLane))
	if a.needsApproval() {
		_, _ = fmt.Fprintf(&buf, "Approval: %s\n", boolLabel(deployReady))
	} else {
		buf.WriteString("Approval: n/a\n")
	}
	if a.needsSecret() {
		_, _ = fmt.Fprintf(
			&buf,
			"%s: %s\n\n",
			a.secretLabel(),
			maskedSecret(secretValue),
		)
	} else {
		buf.WriteString("Secret: n/a\n\n")
	}

	switch stage {
	case formStageProject:
		buf.WriteString("Current prompt: project name\n")
		buf.WriteString(renderInputPrompt("Project name", projectPrompt))
	case formStageProfile:
		buf.WriteString("Current prompt: configured flow\n")
		buf.Write(profilePrompt.Render())
	case formStageLane:
		buf.WriteString("Current prompt: release lane\n")
		buf.Write(lanePrompt.Render())
	case formStageApproval:
		buf.WriteString("Current prompt: deployment approval\n")
		buf.Write(approvePrompt.Render())
	case formStageSecret:
		buf.WriteString("Current prompt: ")
		buf.WriteString(strings.ToLower(a.secretLabel()))
		buf.WriteString("\n")
		buf.Write(secretPrompt.Render())
	case formStageDone:
		buf.WriteString(
			"Form complete. Switch to Flow and press enter to execute the configured pipeline.",
		)
	}

	if plan.Description != "" {
		buf.WriteString("\n\nDescription: ")
		buf.WriteString(plan.Description)
	}

	return buf.String()
}

func renderInputPrompt(label string, prompt *formfx.InputPrompt) string {
	if prompt == nil {
		return label + "\n|"
	}

	value := append([]rune(nil), prompt.Value...)
	cursor := prompt.CursorPos
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(value) {
		cursor = len(value)
	}

	line := string(value[:cursor]) + "|" + string(value[cursor:])
	if line == "|" && len(value) > 0 {
		line = string(value)
	}
	return fmt.Sprintf("%s\n%s", label, line)
}

func (a *app) renderLogsView() string {
	a.mu.Lock()
	lines := append([]string(nil), a.logs...)
	a.mu.Unlock()

	if len(lines) == 0 {
		lines = []string{"No logs yet."}
	}

	return strings.Join([]string{
		"Logs",
		"Press c to clear the log pane.",
		"",
		strings.Join(lines, "\n"),
	}, "\n")
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func maskedSecret(secret string) string {
	if secret == "" {
		return "missing"
	}
	return strings.Repeat("*", len([]rune(secret)))
}
