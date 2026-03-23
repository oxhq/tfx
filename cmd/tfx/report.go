package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type wrapperTaskReport struct {
	Phase           string    `json:"phase,omitempty"`
	ID              string    `json:"id,omitempty"`
	Title           string    `json:"title,omitempty"`
	Status          string    `json:"status,omitempty"`
	Detail          string    `json:"detail,omitempty"`
	Command         []string  `json:"command,omitempty"`
	ContinueOnError bool      `json:"continue_on_error,omitempty"`
	StartedAt       time.Time `json:"started_at,omitempty"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
	DurationMillis  int64     `json:"duration_ms,omitempty"`
}

type wrapperArtifactRecord struct {
	Name        string `json:"name,omitempty"`
	Path        string `json:"path,omitempty"`
	Source      string `json:"source,omitempty"`
	State       string `json:"state,omitempty"`
	Size        string `json:"size,omitempty"`
	Description string `json:"description,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
}

type wrapperArtifactReport struct {
	Phase      string                  `json:"phase,omitempty"`
	StepID     string                  `json:"step_id,omitempty"`
	StepTitle  string                  `json:"step_title,omitempty"`
	Brief      string                  `json:"brief,omitempty"`
	Count      int                     `json:"count,omitempty"`
	Present    int                     `json:"present,omitempty"`
	Missing    int                     `json:"missing,omitempty"`
	Problems   int                     `json:"problems,omitempty"`
	TotalBytes int64                   `json:"total_bytes,omitempty"`
	TotalSize  string                  `json:"total_size,omitempty"`
	Records    []wrapperArtifactRecord `json:"records,omitempty"`
}

type wrapperRunReport struct {
	Status         string                  `json:"status,omitempty"`
	Flow           string                  `json:"flow,omitempty"`
	Lane           string                  `json:"lane,omitempty"`
	Project        string                  `json:"project,omitempty"`
	WorkingDir     string                  `json:"working_dir,omitempty"`
	ConfigPath     string                  `json:"config_path,omitempty"`
	StatePath      string                  `json:"state_path,omitempty"`
	StartedAt      time.Time               `json:"started_at,omitempty"`
	FinishedAt     time.Time               `json:"finished_at,omitempty"`
	DurationMillis int64                   `json:"duration_ms,omitempty"`
	Warnings       int                     `json:"warnings,omitempty"`
	Error          string                  `json:"error,omitempty"`
	Steps          []wrapperTaskReport     `json:"steps,omitempty"`
	Hooks          []wrapperTaskReport     `json:"hooks,omitempty"`
	Artifacts      []wrapperArtifactReport `json:"artifacts,omitempty"`
	RecentRuns     []WrapperRun            `json:"recent_runs,omitempty"`
}

type wrapperSnapshotReport struct {
	Status           string            `json:"status,omitempty"`
	Flow             string            `json:"flow,omitempty"`
	Lane             string            `json:"lane,omitempty"`
	Project          string            `json:"project,omitempty"`
	WorkingDir       string            `json:"working_dir,omitempty"`
	ConfigPath       string            `json:"config_path,omitempty"`
	StatePath        string            `json:"state_path,omitempty"`
	ApprovalRequired bool              `json:"approval_required"`
	Approved         bool              `json:"approved"`
	SecretRequired   bool              `json:"secret_required"`
	SecretProvided   bool              `json:"secret_provided"`
	FlowRunning      bool              `json:"flow_running"`
	FlowDone         bool              `json:"flow_done"`
	Warnings         int               `json:"warnings,omitempty"`
	Error            string            `json:"error,omitempty"`
	LastRun          *wrapperRunReport `json:"last_run,omitempty"`
	RecentRuns       []WrapperRun      `json:"recent_runs,omitempty"`
}

func artifactReportFromSummary(phase string, summary ArtifactSummary) wrapperArtifactReport {
	report := wrapperArtifactReport{
		Phase:      phase,
		StepID:     summary.StepID,
		StepTitle:  summary.StepTitle,
		Brief:      summary.Brief(),
		Count:      summary.Count(),
		Present:    summary.PresentCount(),
		Missing:    summary.MissingCount(),
		Problems:   summary.ProblemCount(),
		TotalBytes: summary.TotalBytes(),
		TotalSize:  FormatArtifactSize(summary.TotalBytes()),
		Records:    make([]wrapperArtifactRecord, 0, len(summary.Records)),
	}
	for _, record := range summary.sortedRecords() {
		report.Records = append(report.Records, wrapperArtifactRecord{
			Name:        record.Name,
			Path:        record.PathLabel(),
			Source:      record.SourceLabel(),
			State:       record.StateLabel(),
			Size:        record.SizeLabel(),
			Description: record.Description,
			Optional:    record.Optional,
		})
	}
	return report
}

func cloneRunReport(report wrapperRunReport) wrapperRunReport {
	cloned := report
	cloned.Steps = append([]wrapperTaskReport(nil), report.Steps...)
	cloned.Hooks = append([]wrapperTaskReport(nil), report.Hooks...)
	cloned.Artifacts = append([]wrapperArtifactReport(nil), report.Artifacts...)
	cloned.RecentRuns = append([]WrapperRun(nil), report.RecentRuns...)
	return cloned
}

func (a *app) snapshotReport() wrapperSnapshotReport {
	a.mu.Lock()
	defer a.mu.Unlock()

	plan := a.currentPlanLocked()
	report := wrapperSnapshotReport{
		Status:           a.flowStateLocked(),
		Flow:             planDisplayName(plan),
		Lane:             a.releaseLane,
		Project:          a.projectName,
		WorkingDir:       plan.WorkingDir,
		ConfigPath:       a.configPath,
		StatePath:        a.statePath,
		ApprovalRequired: plan.RequireApproval,
		Approved:         a.deployReady,
		SecretRequired:   plan.SecretPrompt != "" || plan.SecretEnv != "",
		SecretProvided:   strings.TrimSpace(a.secretValue) != "",
		FlowRunning:      a.flowRunning,
		FlowDone:         a.flowDone,
		Warnings:         a.flowWarnings,
		Error:            a.flowError,
		RecentRuns:       append([]WrapperRun(nil), a.recentRuns...),
	}
	if report.Project == "" {
		report.Project = strings.TrimSpace(a.cfg.Project)
	}
	if !a.lastReport.StartedAt.IsZero() {
		last := cloneRunReport(a.lastReport)
		report.LastRun = &last
	}
	return report
}

func (a *app) jsonSummary() []byte {
	report := a.snapshotReport()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return []byte(fmt.Sprintf("{\"status\":\"error\",\"error\":%q}\n", err.Error()))
	}
	return append(data, '\n')
}

func (a *app) quietSummary() []byte {
	a.mu.Lock()
	report := cloneRunReport(a.lastReport)
	plan := a.currentPlanLocked()
	flowState := a.flowStateLocked()
	projectName := a.projectName
	releaseLane := a.releaseLane
	a.mu.Unlock()

	if projectName == "" {
		projectName = strings.TrimSpace(a.cfg.Project)
	}
	if report.Status == "" {
		report.Status = flowState
		report.Flow = planDisplayName(plan)
		report.Lane = releaseLane
		report.Project = projectName
	}

	line := fmt.Sprintf(
		"status=%s flow=%s lane=%s project=%s warnings=%d artifacts=%d",
		report.Status,
		valueOrDash(report.Flow),
		valueOrDash(report.Lane),
		valueOrDash(report.Project),
		report.Warnings,
		len(report.Artifacts),
	)
	if report.Error != "" {
		line += " error=" + report.Error
	}
	return []byte(line + "\n")
}

func formatRunTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Local().Format(time.RFC3339)
}

func formatRecentRuns(runs []WrapperRun) string {
	if len(runs) == 0 {
		return "none"
	}
	parts := make([]string, 0, min(3, len(runs)))
	for i, run := range runs {
		if i >= 3 {
			break
		}
		parts = append(parts, fmt.Sprintf(
			"%s/%s:%s",
			valueOrDash(run.Flow),
			valueOrDash(run.Lane),
			valueOrDash(run.Status),
		))
	}
	return strings.Join(parts, ", ")
}
