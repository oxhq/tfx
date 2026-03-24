package flowfx

import (
	"context"
	"maps"
)

// Wizard represents a step-by-step interactive flow with structured input/output.
// Each step can access data from previous steps and contribute to shared state.
type Wizard struct {
	steps      []WizardStep
	name       string
	onStart    Hook
	onComplete Hook
	onError    Hook
	state      map[string]any // Shared state between steps
}

// WizardConfig provides configuration for a Wizard flow.
type WizardConfig struct {
	Name         string
	OnStart      Hook
	OnComplete   Hook
	OnError      Hook
	InitialState map[string]any // Initial state for the wizard
}

// DefaultWizardConfig returns the default configuration for a Wizard flow.
func DefaultWizardConfig() WizardConfig {
	return WizardConfig{
		Name:         "wizard",
		InitialState: make(map[string]any),
	}
}

// newWizard creates a new wizard flow with the given configuration.
func newWizard(cfg WizardConfig) *Wizard {
	state := make(map[string]any)
	if cfg.InitialState != nil {
		maps.Copy(state, cfg.InitialState)
	}

	return &Wizard{
		steps:      make([]WizardStep, 0),
		name:       cfg.Name,
		onStart:    cfg.OnStart,
		onComplete: cfg.OnComplete,
		onError:    cfg.OnError,
		state:      state,
	}
}

// --- MULTIPATH API FUNCTIONS ---

// NewWizard creates a new wizard flow with multipath configuration support.
// Supports two usage patterns:
//   - NewWizard()                          // Zero-config, uses defaults
//   - NewWizard(config)                    // Config struct
//
// NewWizard is kept as a zero-ceremony compatibility alias for MustNewWizard.
// Prefer TryNewWizard when args may be user-provided or otherwise fallible.
func NewWizard(args ...any) *Wizard {
	return MustNewWizard(args...)
}

// MustNewWizard creates a new wizard flow and panics on invalid multipath input.
func MustNewWizard(args ...any) *Wizard {
	return mustValue(TryNewWizard(args...))
}

// TryNewWizard creates a new wizard flow without panicking on invalid args.
func TryNewWizard(args ...any) (*Wizard, error) {
	return tryConstruct(args, DefaultWizardConfig(), newWizard)
}

// New creates a new WizardBuilder for DSL chaining.
func New() *WizardBuilder {
	return &WizardBuilder{config: DefaultWizardConfig()}
}

// AddStep appends a step to the wizard flow.
func (w *Wizard) AddStep(step WizardStep) *Wizard {
	w.steps = append(w.steps, step)
	return w
}

// AddFunc is a convenience method to add a function as a wizard step.
func (w *Wizard) AddFunc(
	label string,
	fn func(ctx context.Context, input map[string]any) (map[string]any, error),
) *Wizard {
	step := NewWizardStepFunc(label, fn)
	w.steps = append(w.steps, step)
	return w
}

// Run executes all steps sequentially, maintaining shared state.
// It implements the Flow interface.
func (w *Wizard) Run(ctx context.Context) error {
	if len(w.steps) == 0 {
		return NewFlowError(w.name, "", ErrEmptyFlow)
	}

	// Call onStart hook if provided
	if w.onStart != nil {
		w.onStart(ctx, w.name, nil)
	}

	// Execute each step sequentially
	for _, step := range w.steps {
		select {
		case <-ctx.Done():
			err := NewFlowError(w.name, step.Label(), ErrCanceled)
			if w.onError != nil {
				w.onError(ctx, w.name, err)
			}
			return err
		default:
		}

		// Execute the step with current state
		output, err := step.Execute(ctx, w.state)
		if err != nil {
			flowErr := NewFlowError(w.name, step.Label(), err)
			if w.onError != nil {
				w.onError(ctx, w.name, flowErr)
			}
			return flowErr
		}

		// Merge output into state
		maps.Copy(w.state, output)
	}

	// All steps completed successfully
	if w.onComplete != nil {
		w.onComplete(ctx, w.name, nil)
	}

	return nil
}

// GetState returns a copy of the current wizard state.
func (w *Wizard) GetState() map[string]any {
	state := make(map[string]any)
	maps.Copy(state, w.state)
	return state
}

// SetState sets a value in the wizard state.
func (w *Wizard) SetState(key string, value any) *Wizard {
	w.state[key] = value
	return w
}

// Steps returns a copy of the steps in the wizard flow.
func (w *Wizard) Steps() []WizardStep {
	steps := make([]WizardStep, len(w.steps))
	copy(steps, w.steps)
	return steps
}

// Len returns the number of steps in the wizard flow.
func (w *Wizard) Len() int {
	return len(w.steps)
}

// --- DSL BUILDER ---

// WizardBuilder provides a fluent API for building wizard flows.
type WizardBuilder struct {
	config WizardConfig
	steps  []WizardStep
}

// Name sets the name of the wizard flow.
func (wb *WizardBuilder) Name(name string) *WizardBuilder {
	wb.config.Name = name
	return wb
}

// OnStart sets the start hook.
func (wb *WizardBuilder) OnStart(hook Hook) *WizardBuilder {
	wb.config.OnStart = hook
	return wb
}

// OnComplete sets the complete hook.
func (wb *WizardBuilder) OnComplete(hook Hook) *WizardBuilder {
	wb.config.OnComplete = hook
	return wb
}

// OnError sets the error hook.
func (wb *WizardBuilder) OnError(hook Hook) *WizardBuilder {
	wb.config.OnError = hook
	return wb
}

// InitialState sets the initial state for the wizard.
func (wb *WizardBuilder) InitialState(state map[string]any) *WizardBuilder {
	wb.config.InitialState = state
	return wb
}

// Step adds a step to the wizard flow.
func (wb *WizardBuilder) Step(step WizardStep) *WizardBuilder {
	wb.steps = append(wb.steps, step)
	return wb
}

// Func adds a function as a wizard step.
func (wb *WizardBuilder) Func(
	label string,
	fn func(ctx context.Context, input map[string]any) (map[string]any, error),
) *WizardBuilder {
	step := NewWizardStepFunc(label, fn)
	wb.steps = append(wb.steps, step)
	return wb
}

// Build creates a new Wizard instance without running it.
func (wb *WizardBuilder) Build() *Wizard {
	wizard := newWizard(wb.config)
	wizard.steps = cloneSlice(wb.steps)
	return wizard
}

// Run creates and runs the wizard flow.
func (wb *WizardBuilder) Run(ctx context.Context) error {
	return wb.Build().Run(ctx)
}
