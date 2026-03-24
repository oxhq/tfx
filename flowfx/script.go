package flowfx

import (
	"context"
	"fmt"
	"strings"
)

// Script represents a scripted flow that executes steps sequentially with enhanced logging.
// It provides script-like execution with detailed traceability and error reporting.
type Script struct {
	steps      []ScriptStep
	name       string
	onStart    Hook
	onComplete Hook
	onError    Hook
	logger     ScriptLogger // Optional logger for enhanced traceability
}

// ScriptStep represents a single step in a script flow with metadata.
type ScriptStep struct {
	Name        string // Step name for logging
	Description string // Optional description
	Step        Step   // The actual step to execute
	Critical    bool   // If true, failure stops execution
	Silent      bool   // If true, don't log this step
}

// ScriptLogger defines the interface for script execution logging.
// This allows integration with external logging systems without importing specific implementations.
type ScriptLogger interface {
	LogStepStart(name, description string)
	LogStepComplete(name string)
	LogStepError(name string, err error)
	LogScriptStart(name string)
	LogScriptComplete(name string)
	LogScriptError(name string, err error)
}

// ScriptConfig provides configuration for a Script flow.
type ScriptConfig struct {
	Name       string
	OnStart    Hook
	OnComplete Hook
	OnError    Hook
	Logger     ScriptLogger
}

// DefaultScriptConfig returns the default configuration for a Script flow.
func DefaultScriptConfig() ScriptConfig {
	return ScriptConfig{
		Name: "script",
	}
}

// newScript creates a new script flow with the given configuration.
func newScript(cfg ScriptConfig) *Script {
	return &Script{
		steps:      make([]ScriptStep, 0),
		name:       cfg.Name,
		onStart:    cfg.OnStart,
		onComplete: cfg.OnComplete,
		onError:    cfg.OnError,
		logger:     cfg.Logger,
	}
}

// --- MULTIPATH API FUNCTIONS ---

// NewScript creates a new script flow with multipath configuration support.
// Supports two usage patterns:
//   - NewScript()                          // Zero-config, uses defaults
//   - NewScript(config)                    // Config struct
//
// NewScript is kept as a zero-ceremony compatibility alias for MustNewScript.
// Prefer TryNewScript when args may be user-provided or otherwise fallible.
func NewScript(args ...any) *Script {
	return MustNewScript(args...)
}

// MustNewScript creates a new script flow and panics on invalid multipath
// input.
func MustNewScript(args ...any) *Script {
	return mustValue(TryNewScript(args...))
}

// TryNewScript creates a new script flow without panicking on invalid args.
func TryNewScript(args ...any) (*Script, error) {
	return tryConstruct(args, DefaultScriptConfig(), newScript)
}

// NewScriptBuilder creates a new ScriptBuilder for DSL chaining.
func NewScriptBuilder() *ScriptBuilder {
	return &ScriptBuilder{config: DefaultScriptConfig()}
}

// AddStep appends a step to the script flow.
func (s *Script) AddStep(scriptStep ScriptStep) *Script {
	s.steps = append(s.steps, scriptStep)
	return s
}

// AddSimpleStep is a convenience method to add a simple step.
func (s *Script) AddSimpleStep(name string, step Step) *Script {
	scriptStep := ScriptStep{
		Name: name,
		Step: step,
	}
	s.steps = append(s.steps, scriptStep)
	return s
}

// AddTask is a convenience method to add a Task as a script step.
func (s *Script) AddTask(name string, task *Task) *Script {
	scriptStep := ScriptStep{
		Name: name,
		Step: task,
	}
	s.steps = append(s.steps, scriptStep)
	return s
}

// AddFunc is a convenience method to add a function as a script step.
func (s *Script) AddFunc(name, label string, fn func(ctx context.Context) error) *Script {
	task := NewTask(label, fn)
	scriptStep := ScriptStep{
		Name: name,
		Step: task,
	}
	s.steps = append(s.steps, scriptStep)
	return s
}

// AddCriticalStep adds a critical step that will stop execution if it fails.
func (s *Script) AddCriticalStep(name string, step Step) *Script {
	scriptStep := ScriptStep{
		Name:     name,
		Step:     step,
		Critical: true,
	}
	s.steps = append(s.steps, scriptStep)
	return s
}

// Run executes all steps sequentially with enhanced logging and error traceability.
// It implements the Flow interface.
func (s *Script) Run(ctx context.Context) error {
	if len(s.steps) == 0 {
		return NewFlowError(s.name, "", ErrEmptyFlow)
	}

	// Call onStart hook if provided
	if s.onStart != nil {
		s.onStart(ctx, s.name, nil)
	}

	// Log script start
	if s.logger != nil {
		s.logger.LogScriptStart(s.name)
	}

	var scriptErrors []error

	// Execute each step sequentially
	for i, scriptStep := range s.steps {
		select {
		case <-ctx.Done():
			err := NewFlowError(s.name, scriptStep.Name, ErrCanceled)
			if s.onError != nil {
				s.onError(ctx, s.name, err)
			}
			if s.logger != nil {
				s.logger.LogScriptError(s.name, err)
			}
			return err
		default:
		}

		// Log step start
		if s.logger != nil && !scriptStep.Silent {
			s.logger.LogStepStart(scriptStep.Name, scriptStep.Description)
		}

		// Execute the step
		if err := scriptStep.Step.Execute(ctx); err != nil {
			stepName := scriptStep.Name
			if stepName == "" {
				stepName = fmt.Sprintf("step_%d", i+1)
			}

			flowErr := NewFlowError(s.name, stepName, err)
			scriptErrors = append(scriptErrors, flowErr)

			// Log step error
			if s.logger != nil && !scriptStep.Silent {
				s.logger.LogStepError(scriptStep.Name, err)
			}

			// If this is a critical step, stop execution immediately
			if scriptStep.Critical {
				if s.onError != nil {
					s.onError(ctx, s.name, flowErr)
				}
				if s.logger != nil {
					s.logger.LogScriptError(s.name, flowErr)
				}
				return flowErr
			}

			// Continue with non-critical step errors
			continue
		}

		// Log step completion
		if s.logger != nil && !scriptStep.Silent {
			s.logger.LogStepComplete(scriptStep.Name)
		}
	}

	// Check if we have any accumulated errors
	if len(scriptErrors) > 0 {
		multiErr := NewMultiError()
		for _, err := range scriptErrors {
			multiErr.Add(err)
		}

		scriptErr := multiErr.ToError()
		if s.onError != nil {
			s.onError(ctx, s.name, scriptErr)
		}
		if s.logger != nil {
			s.logger.LogScriptError(s.name, scriptErr)
		}
		return scriptErr
	}

	// All steps completed successfully
	if s.onComplete != nil {
		s.onComplete(ctx, s.name, nil)
	}

	// Log script completion
	if s.logger != nil {
		s.logger.LogScriptComplete(s.name)
	}

	return nil
}

// Steps returns a copy of the script steps.
func (s *Script) Steps() []ScriptStep {
	steps := make([]ScriptStep, len(s.steps))
	copy(steps, s.steps)
	return steps
}

// Len returns the number of steps in the script flow.
func (s *Script) Len() int {
	return len(s.steps)
}

// Summary returns a string summary of the script.
func (s *Script) Summary() string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "Script: %s (%d steps)\n", s.name, len(s.steps))
	for i, step := range s.steps {
		status := "normal"
		if step.Critical {
			status = "critical"
		}
		if step.Silent {
			status += ", silent"
		}
		_, _ = fmt.Fprintf(&sb, "  %d. %s (%s)", i+1, step.Name, status)
		if step.Description != "" {
			_, _ = fmt.Fprintf(&sb, " - %s", step.Description)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// --- DSL BUILDER ---

// ScriptBuilder provides a fluent API for building script flows.
type ScriptBuilder struct {
	config ScriptConfig
	steps  []ScriptStep
}

// Name sets the name of the script flow.
func (sb *ScriptBuilder) Name(name string) *ScriptBuilder {
	sb.config.Name = name
	return sb
}

// OnStart sets the start hook.
func (sb *ScriptBuilder) OnStart(hook Hook) *ScriptBuilder {
	sb.config.OnStart = hook
	return sb
}

// OnComplete sets the complete hook.
func (sb *ScriptBuilder) OnComplete(hook Hook) *ScriptBuilder {
	sb.config.OnComplete = hook
	return sb
}

// OnError sets the error hook.
func (sb *ScriptBuilder) OnError(hook Hook) *ScriptBuilder {
	sb.config.OnError = hook
	return sb
}

// Logger sets the script logger.
func (sb *ScriptBuilder) Logger(logger ScriptLogger) *ScriptBuilder {
	sb.config.Logger = logger
	return sb
}

// Step adds a script step to the flow.
func (sb *ScriptBuilder) Step(scriptStep ScriptStep) *ScriptBuilder {
	sb.steps = append(sb.steps, scriptStep)
	return sb
}

// SimpleStep adds a simple step with name and step.
func (sb *ScriptBuilder) SimpleStep(name string, step Step) *ScriptBuilder {
	scriptStep := ScriptStep{
		Name: name,
		Step: step,
	}
	sb.steps = append(sb.steps, scriptStep)
	return sb
}

// Task adds a task as a script step.
func (sb *ScriptBuilder) Task(name string, task *Task) *ScriptBuilder {
	scriptStep := ScriptStep{
		Name: name,
		Step: task,
	}
	sb.steps = append(sb.steps, scriptStep)
	return sb
}

// Func adds a function as a script step.
func (sb *ScriptBuilder) Func(
	name, label string,
	fn func(ctx context.Context) error,
) *ScriptBuilder {
	task := NewTask(label, fn)
	scriptStep := ScriptStep{
		Name: name,
		Step: task,
	}
	sb.steps = append(sb.steps, scriptStep)
	return sb
}

// CriticalStep adds a critical step that will stop execution if it fails.
func (sb *ScriptBuilder) CriticalStep(name string, step Step) *ScriptBuilder {
	scriptStep := ScriptStep{
		Name:     name,
		Step:     step,
		Critical: true,
	}
	sb.steps = append(sb.steps, scriptStep)
	return sb
}

// SilentStep adds a step that won't be logged.
func (sb *ScriptBuilder) SilentStep(name string, step Step) *ScriptBuilder {
	scriptStep := ScriptStep{
		Name:   name,
		Step:   step,
		Silent: true,
	}
	sb.steps = append(sb.steps, scriptStep)
	return sb
}

// Build creates a new Script instance without running it.
func (sb *ScriptBuilder) Build() *Script {
	script := newScript(sb.config)
	script.steps = cloneSlice(sb.steps)
	return script
}

// Run creates and runs the script flow.
func (sb *ScriptBuilder) Run(ctx context.Context) error {
	return sb.Build().Run(ctx)
}
