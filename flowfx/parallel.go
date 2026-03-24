package flowfx

import (
	"context"
	"fmt"
	"sync"
)

// Parallel represents a parallel flow that executes steps concurrently.
// All steps run simultaneously, and the flow waits for all to complete.
type Parallel struct {
	steps      []Step
	name       string
	onStart    Hook
	onComplete Hook
	onError    Hook
	failFast   bool // If true, cancel all steps when one fails
}

// ParallelConfig provides configuration for a Parallel flow.
type ParallelConfig struct {
	Name       string
	OnStart    Hook
	OnComplete Hook
	OnError    Hook
	FailFast   bool
}

// DefaultParallelConfig returns the default configuration for a Parallel flow.
func DefaultParallelConfig() ParallelConfig {
	return ParallelConfig{
		Name:     "parallel",
		FailFast: false,
	}
}

// newParallel creates a new parallel flow with the given configuration.
func newParallel(cfg ParallelConfig) *Parallel {
	return &Parallel{
		steps:      make([]Step, 0),
		name:       cfg.Name,
		onStart:    cfg.OnStart,
		onComplete: cfg.OnComplete,
		onError:    cfg.OnError,
		failFast:   cfg.FailFast,
	}
}

// --- MULTIPATH API FUNCTIONS ---

// NewParallel creates a new parallel flow with multipath configuration support.
// Supports two usage patterns:
//   - NewParallel()                          // Zero-config, uses defaults
//   - NewParallel(config)                    // Config struct
//
// NewParallel is kept as a zero-ceremony compatibility alias for
// MustNewParallel. Prefer TryNewParallel when args may be user-provided or
// otherwise fallible.
func NewParallel(args ...any) *Parallel {
	return MustNewParallel(args...)
}

// MustNewParallel creates a new parallel flow and panics on invalid multipath
// input.
func MustNewParallel(args ...any) *Parallel {
	return mustValue(TryNewParallel(args...))
}

// TryNewParallel creates a new parallel flow without panicking on invalid args.
func TryNewParallel(args ...any) (*Parallel, error) {
	return tryConstruct(args, DefaultParallelConfig(), newParallel)
}

// NewParallelBuilder creates a new ParallelBuilder for DSL chaining.
func NewParallelBuilder() *ParallelBuilder {
	return &ParallelBuilder{config: DefaultParallelConfig()}
}

// Add appends a step to the parallel flow.
func (p *Parallel) Add(step Step) *Parallel {
	p.steps = append(p.steps, step)
	return p
}

// AddTask is a convenience method to add a Task as a step.
func (p *Parallel) AddTask(task *Task) *Parallel {
	p.steps = append(p.steps, task)
	return p
}

// AddFunc is a convenience method to add a function as a step.
func (p *Parallel) AddFunc(label string, fn func(ctx context.Context) error) *Parallel {
	task := NewTask(label, fn)
	p.steps = append(p.steps, task)
	return p
}

// Run executes all steps in parallel and waits for completion.
// It implements the Flow interface.
func (p *Parallel) Run(ctx context.Context) error {
	if len(p.steps) == 0 {
		return NewFlowError(p.name, "", ErrEmptyFlow)
	}

	// Call onStart hook if provided
	if p.onStart != nil {
		p.onStart(ctx, p.name, nil)
	}

	// Create context for cancellation in fail-fast mode
	execCtx := ctx
	var cancel context.CancelFunc
	if p.failFast {
		execCtx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	// Channel to collect errors from goroutines
	errCh := make(chan error, len(p.steps))
	var wg sync.WaitGroup

	// Start all steps concurrently
	for i, step := range p.steps {
		wg.Add(1)
		go func(stepIndex int, s Step) {
			defer wg.Done()

			// Execute the step
			if err := s.Execute(execCtx); err != nil {
				stepName := fmt.Sprintf("step_%d", stepIndex+1)
				if task, ok := s.(*Task); ok && task.Label != "" {
					stepName = task.Label
				}

				flowErr := NewFlowError(p.name, stepName, err)
				errCh <- flowErr

				// Cancel other steps if fail-fast is enabled
				if p.failFast && cancel != nil {
					cancel()
				}
				return
			}

			// Send nil to indicate success
			errCh <- nil
		}(i, step)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect results
	multiErr := NewMultiError()
	successCount := 0

	for err := range errCh {
		if err != nil {
			multiErr.Add(err)
		} else {
			successCount++
		}
	}

	// Handle results
	if multiErr.HasErrors() {
		if p.onError != nil {
			p.onError(ctx, p.name, multiErr.ToError())
		}
		return multiErr.ToError()
	}

	// All steps completed successfully
	if p.onComplete != nil {
		p.onComplete(ctx, p.name, nil)
	}

	return nil
}

// Steps returns a copy of the steps in the parallel flow.
func (p *Parallel) Steps() []Step {
	steps := make([]Step, len(p.steps))
	copy(steps, p.steps)
	return steps
}

// Len returns the number of steps in the parallel flow.
func (p *Parallel) Len() int {
	return len(p.steps)
}

// --- DSL BUILDER ---

// ParallelBuilder provides a fluent API for building parallel flows.
type ParallelBuilder struct {
	config ParallelConfig
	steps  []Step
}

// Name sets the name of the parallel flow.
func (pb *ParallelBuilder) Name(name string) *ParallelBuilder {
	pb.config.Name = name
	return pb
}

// OnStart sets the start hook.
func (pb *ParallelBuilder) OnStart(hook Hook) *ParallelBuilder {
	pb.config.OnStart = hook
	return pb
}

// OnComplete sets the complete hook.
func (pb *ParallelBuilder) OnComplete(hook Hook) *ParallelBuilder {
	pb.config.OnComplete = hook
	return pb
}

// OnError sets the error hook.
func (pb *ParallelBuilder) OnError(hook Hook) *ParallelBuilder {
	pb.config.OnError = hook
	return pb
}

// FailFast enables fail-fast mode.
func (pb *ParallelBuilder) FailFast(enabled bool) *ParallelBuilder {
	pb.config.FailFast = enabled
	return pb
}

// Step adds a step to the parallel flow.
func (pb *ParallelBuilder) Step(step Step) *ParallelBuilder {
	pb.steps = append(pb.steps, step)
	return pb
}

// Steps adds multiple steps to the parallel flow.
func (pb *ParallelBuilder) Steps(steps ...Step) *ParallelBuilder {
	pb.steps = append(pb.steps, steps...)
	return pb
}

// Task adds a task as a step to the parallel flow.
func (pb *ParallelBuilder) Task(task *Task) *ParallelBuilder {
	pb.steps = append(pb.steps, task)
	return pb
}

// Func adds a function as a step to the parallel flow.
func (pb *ParallelBuilder) Func(label string, fn func(ctx context.Context) error) *ParallelBuilder {
	task := NewTask(label, fn)
	pb.steps = append(pb.steps, task)
	return pb
}

// Build creates a new Parallel instance without running it.
func (pb *ParallelBuilder) Build() *Parallel {
	parallel := newParallel(pb.config)
	parallel.steps = cloneSlice(pb.steps)
	return parallel
}

// Run creates and runs the parallel flow.
func (pb *ParallelBuilder) Run(ctx context.Context) error {
	return pb.Build().Run(ctx)
}
