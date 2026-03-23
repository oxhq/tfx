package flowfx

import (
	"context"
	"time"
)

// Step represents a single executable unit within a flow.
// Steps are the fundamental building blocks of all flow types.
type Step interface {
	Execute(ctx context.Context) error
}

// Flow represents an executable workflow that can contain multiple steps.
// All flow types (Sequence, Parallel, Branch, etc.) implement this interface.
type Flow interface {
	Run(ctx context.Context) error
}

// ProgressReporter defines the interface for reporting task progress.
// This interface allows integration with external progress systems without
// importing specific implementations.
type ProgressReporter interface {
	// Start begins progress reporting for a task
	Start(label string, total int)
	// Update reports current progress
	Update(current int)
	// Complete marks the task as finished
	Complete()
	// Error reports that the task failed
	Error(err error)
}

// Condition represents a conditional function for branching logic.
// It receives context and returns true if the condition is met.
type Condition func(ctx context.Context) bool

// StepFunc is a function that can be used as a Step.
type StepFunc func(ctx context.Context) error

// Execute implements the Step interface for StepFunc.
func (f StepFunc) Execute(ctx context.Context) error {
	return f(ctx)
}

// Runner provides explicit control over flow execution lifecycle.
// It manages continuous execution and responsive cancellation.
type Runner interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// WizardStep represents a step in a wizard flow with structured input/output.
type WizardStep interface {
	Execute(ctx context.Context, input map[string]any) (output map[string]any, err error)
	Label() string
}

// WizardStepFunc is a function adapter for WizardStep.
type WizardStepFunc struct {
	label string
	fn    func(ctx context.Context, input map[string]any) (map[string]any, error)
}

// NewWizardStepFunc creates a new WizardStepFunc.
func NewWizardStepFunc(
	label string,
	fn func(ctx context.Context, input map[string]any) (map[string]any, error),
) *WizardStepFunc {
	return &WizardStepFunc{label: label, fn: fn}
}

// Execute implements WizardStep.
func (w *WizardStepFunc) Execute(
	ctx context.Context,
	input map[string]any,
) (map[string]any, error) {
	return w.fn(ctx, input)
}

// Label implements WizardStep.
func (w *WizardStepFunc) Label() string {
	return w.label
}

// Hook defines callback functions for flow lifecycle events.
type Hook func(ctx context.Context, label string, err error)

// RetryConfig defines retry behavior for tasks.
type RetryConfig struct {
	MaxAttempts int
	Delay       time.Duration
	Backoff     float64 // Multiplier for exponential backoff
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		Delay:       100 * time.Millisecond,
		Backoff:     2.0,
	}
}
