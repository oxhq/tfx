package progrefx

import (
	"fmt"
	"strings"
	"sync"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/runfx"
	"github.com/oxhq/tfx/terminal"
)

// StepStatus describes the lifecycle state of a step in the stepper view.
type StepStatus int

const (
	StepPending StepStatus = iota
	StepActive
	StepDone
	StepFailed
	StepSkipped
)

// StepItem represents a single step in a stepper flow.
type StepItem struct {
	Title  string
	Detail string
	Status StepStatus
}

// StepperView renders a list of ordered steps with colored statuses.
type StepperView struct {
	label    string
	steps    []StepItem
	theme    ProgressTheme
	detector *terminal.Detector
	isTTY    bool

	mu sync.Mutex
}

func newStepper(cfg StepperConfig) *StepperView {
	detect := cfg.DetectTTY
	if detect == nil {
		detect = runfx.DetectTTY
	}
	theme := cfg.Theme
	if theme == (ProgressTheme{}) {
		theme = MaterialTheme
	}
	tty := detect()

	stepper := &StepperView{
		label:    cfg.Label,
		steps:    cloneSteps(cfg.Steps),
		theme:    theme,
		detector: terminal.NewDetector(cfg.Writer),
		isTTY:    tty.IsTTY,
	}
	stepper.ensureActiveStep()
	return stepper
}

func cloneSteps(steps []StepItem) []StepItem {
	cloned := make([]StepItem, len(steps))
	copy(cloned, steps)
	return cloned
}

func (s *StepperView) ensureActiveStep() {
	if len(s.steps) == 0 {
		return
	}
	for _, step := range s.steps {
		if step.Status == StepActive {
			return
		}
	}
	for i := range s.steps {
		if s.steps[i].Status == StepPending {
			s.steps[i].Status = StepActive
			return
		}
	}
}

func (s *StepperView) statusColor(status StepStatus) color.Color {
	switch status {
	case StepDone:
		return s.theme.CompleteColor
	case StepActive:
		return s.theme.PercentColor
	case StepFailed:
		return color.MaterialRed
	case StepSkipped:
		return s.theme.BorderColor
	default:
		return s.theme.IncompleteColor
	}
}

func (s *StepperView) statusSymbol(status StepStatus) string {
	if !s.isTTY {
		switch status {
		case StepDone:
			return "[done]"
		case StepActive:
			return "[now ]"
		case StepFailed:
			return "[fail]"
		case StepSkipped:
			return "[skip]"
		default:
			return "[todo]"
		}
	}

	switch status {
	case StepDone:
		return "✓"
	case StepActive:
		return "→"
	case StepFailed:
		return "✗"
	case StepSkipped:
		return "•"
	default:
		return "·"
	}
}

func (s *StepperView) style(text string, c color.Color) string {
	if !s.isTTY {
		return text
	}
	return s.theme.RenderColor(c, s.detector) + text + color.Reset
}

// Render returns the full multi-line stepper representation.
func (s *StepperView) Render() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lines []string
	if s.label != "" {
		lines = append(lines, s.style(s.label, s.theme.LabelColor))
	}

	for _, step := range s.steps {
		line := fmt.Sprintf(
			"%s %s",
			s.style(s.statusSymbol(step.Status), s.statusColor(step.Status)),
			step.Title,
		)
		if step.Detail != "" {
			line += " - " + step.Detail
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Steps returns a copy of the configured steps.
func (s *StepperView) Steps() []StepItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneSteps(s.steps)
}

// SetLabel updates the stepper title.
func (s *StepperView) SetLabel(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.label = label
}

// SetSteps replaces the whole step list.
func (s *StepperView) SetSteps(steps []StepItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = cloneSteps(steps)
	s.ensureActiveStep()
}

// SetStatus updates the status of a specific step.
func (s *StepperView) SetStatus(index int, status StepStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.steps) {
		return
	}
	if status == StepActive {
		for i := range s.steps {
			if i != index && s.steps[i].Status == StepActive {
				s.steps[i].Status = StepPending
			}
		}
	}
	s.steps[index].Status = status
}

// SetDetail updates the detail of a specific step.
func (s *StepperView) SetDetail(index int, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.steps) {
		return
	}
	s.steps[index].Detail = detail
}

// Activate marks a step as the currently active one.
func (s *StepperView) Activate(index int) {
	s.SetStatus(index, StepActive)
}

// Complete marks a step as completed.
func (s *StepperView) Complete(index int) {
	s.SetStatus(index, StepDone)
}

// Fail marks a step as failed and optionally updates its detail.
func (s *StepperView) Fail(index int, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.steps) {
		return
	}
	s.steps[index].Status = StepFailed
	if detail != "" {
		s.steps[index].Detail = detail
	}
}

// Skip marks a step as skipped.
func (s *StepperView) Skip(index int) {
	s.SetStatus(index, StepSkipped)
}

// Advance completes the active step and activates the next pending one.
func (s *StepperView) Advance() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.steps) == 0 {
		return
	}

	active := -1
	for i := range s.steps {
		if s.steps[i].Status == StepActive {
			active = i
			break
		}
	}

	if active == -1 {
		s.ensureActiveStep()
		return
	}

	s.steps[active].Status = StepDone
	for i := active + 1; i < len(s.steps); i++ {
		if s.steps[i].Status == StepPending {
			s.steps[i].Status = StepActive
			return
		}
	}
}
