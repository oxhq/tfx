package flowfx

import (
	"context"
	"fmt"
)

// Sequence represents a sequential flow that executes steps one after another.
// If any step fails, the sequence stops and returns the error.
type Sequence struct {
	steps      []Step
	name       string
	onStart    Hook
	onComplete Hook
	onError    Hook
}

// SequenceConfig provides configuration for a Sequence.
type SequenceConfig struct {
	Name       string
	OnStart    Hook
	OnComplete Hook
	OnError    Hook
}

// DefaultSequenceConfig returns the default configuration for a Sequence.
func DefaultSequenceConfig() SequenceConfig {
	return SequenceConfig{
		Name: "sequence",
	}
}

// newSequence creates a new sequence with the given configuration.
func newSequence(cfg SequenceConfig) *Sequence {
	return &Sequence{
		steps:      make([]Step, 0),
		name:       cfg.Name,
		onStart:    cfg.OnStart,
		onComplete: cfg.OnComplete,
		onError:    cfg.OnError,
	}
}

// --- MULTIPATH API FUNCTIONS ---

// NewSequence creates a new sequential flow with multipath configuration support.
// Supports two usage patterns:
//   - NewSequence()                          // Zero-config, uses defaults
//   - NewSequence(config)                    // Config struct
//
// NewSequence is kept as a zero-ceremony compatibility alias for
// MustNewSequence. Prefer TryNewSequence when args may be user-provided or
// otherwise fallible.
func NewSequence(args ...any) *Sequence {
	return MustNewSequence(args...)
}

// MustNewSequence creates a new sequential flow and panics on invalid
// multipath input.
func MustNewSequence(args ...any) *Sequence {
	return mustValue(TryNewSequence(args...))
}

// TryNewSequence creates a new sequential flow without panicking on invalid args.
func TryNewSequence(args ...any) (*Sequence, error) {
	return tryConstruct(args, DefaultSequenceConfig(), newSequence)
}

// NewSequenceBuilder creates a new SequenceBuilder for DSL chaining.
func NewSequenceBuilder() *SequenceBuilder {
	return &SequenceBuilder{config: DefaultSequenceConfig()}
}

// Add appends a step to the sequence.
func (s *Sequence) Add(step Step) *Sequence {
	s.steps = append(s.steps, step)
	return s
}

// AddTask is a convenience method to add a Task as a step.
func (s *Sequence) AddTask(task *Task) *Sequence {
	s.steps = append(s.steps, task)
	return s
}

// AddFunc is a convenience method to add a function as a step.
func (s *Sequence) AddFunc(label string, fn func(ctx context.Context) error) *Sequence {
	task := NewTask(label, fn)
	s.steps = append(s.steps, task)
	return s
}

// Run executes all steps in the sequence sequentially.
// It implements the Flow interface.
func (s *Sequence) Run(ctx context.Context) error {
	if len(s.steps) == 0 {
		return NewFlowError(s.name, "", ErrEmptyFlow)
	}

	// Call onStart hook if provided
	if s.onStart != nil {
		s.onStart(ctx, s.name, nil)
	}

	// Execute each step sequentially
	for i, step := range s.steps {
		// Check for cancellation before each step
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if s.onError != nil {
				s.onError(ctx, s.name, err)
			}
			return NewFlowError(s.name, fmt.Sprintf("step_%d", i+1), err)
		default:
		}

		// Execute the step
		if err := step.Execute(ctx); err != nil {
			if s.onError != nil {
				s.onError(ctx, s.name, err)
			}

			stepName := fmt.Sprintf("step_%d", i+1)
			if task, ok := step.(*Task); ok && task.Label != "" {
				stepName = task.Label
			}

			return NewFlowError(s.name, stepName, err)
		}
	}

	// Call onComplete hook if provided
	if s.onComplete != nil {
		s.onComplete(ctx, s.name, nil)
	}

	return nil
}

// Steps returns a copy of the steps in the sequence.
func (s *Sequence) Steps() []Step {
	steps := make([]Step, len(s.steps))
	copy(steps, s.steps)
	return steps
}

// Len returns the number of steps in the sequence.
func (s *Sequence) Len() int {
	return len(s.steps)
}

// --- DSL BUILDER ---

// SequenceBuilder provides a fluent API for building sequences.
type SequenceBuilder struct {
	config SequenceConfig
	steps  []Step
}

// Name sets the name of the sequence.
func (sb *SequenceBuilder) Name(name string) *SequenceBuilder {
	sb.config.Name = name
	return sb
}

// OnStart sets the start hook.
func (sb *SequenceBuilder) OnStart(hook Hook) *SequenceBuilder {
	sb.config.OnStart = hook
	return sb
}

// OnComplete sets the complete hook.
func (sb *SequenceBuilder) OnComplete(hook Hook) *SequenceBuilder {
	sb.config.OnComplete = hook
	return sb
}

// OnError sets the error hook.
func (sb *SequenceBuilder) OnError(hook Hook) *SequenceBuilder {
	sb.config.OnError = hook
	return sb
}

// Step adds a step to the sequence.
func (sb *SequenceBuilder) Step(step Step) *SequenceBuilder {
	sb.steps = append(sb.steps, step)
	return sb
}

// Steps adds multiple steps to the sequence.
func (sb *SequenceBuilder) Steps(steps ...Step) *SequenceBuilder {
	sb.steps = append(sb.steps, steps...)
	return sb
}

// Task adds a task as a step to the sequence.
func (sb *SequenceBuilder) Task(task *Task) *SequenceBuilder {
	sb.steps = append(sb.steps, task)
	return sb
}

// Func adds a function as a step to the sequence.
func (sb *SequenceBuilder) Func(label string, fn func(ctx context.Context) error) *SequenceBuilder {
	task := NewTask(label, fn)
	sb.steps = append(sb.steps, task)
	return sb
}

// Build creates a new Sequence instance without running it.
func (sb *SequenceBuilder) Build() *Sequence {
	seq := newSequence(sb.config)
	seq.steps = cloneSlice(sb.steps)
	return seq
}

// Run creates and runs the sequence.
func (sb *SequenceBuilder) Run(ctx context.Context) error {
	return sb.Build().Run(ctx)
}
