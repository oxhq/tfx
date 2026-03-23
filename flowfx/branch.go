package flowfx

import (
	"context"
	"fmt"
)

// Branch represents a conditional flow that executes different paths based on a condition.
type Branch struct {
	condition  func(ctx context.Context) (bool, error)
	truePath   Flow
	falsePath  Flow
	name       string
	onStart    Hook
	onComplete Hook
	onError    Hook
}

// BranchConfig provides configuration for a Branch.
type BranchConfig struct {
	Name       string
	OnStart    Hook
	OnComplete Hook
	OnError    Hook
}

// DefaultBranchConfig returns the default configuration for a Branch.
func DefaultBranchConfig() BranchConfig {
	return BranchConfig{
		Name: "branch",
	}
}

// newBranch creates a new branch with the given configuration and condition.
func newBranch(condition func(ctx context.Context) (bool, error), cfg BranchConfig) *Branch {
	if condition == nil {
		condition = func(ctx context.Context) (bool, error) {
			return false, ErrInvalidCondition
		}
	}

	return &Branch{
		condition:  condition,
		name:       cfg.Name,
		onStart:    cfg.OnStart,
		onComplete: cfg.OnComplete,
		onError:    cfg.OnError,
	}
}

// --- MULTIPATH API FUNCTIONS ---

// NewBranch creates a new conditional branch flow with multipath configuration support.
// Supports two usage patterns:
//   - NewBranch(condition)                          // Zero-config, uses defaults
//   - NewBranch(condition, config)                  // Config struct
func NewBranch(condition func(ctx context.Context) (bool, error), args ...any) *Branch {
	return must(TryNewBranch(condition, args...))
}

// TryNewBranch creates a new branch flow without panicking on invalid args.
func TryNewBranch(
	condition func(ctx context.Context) (bool, error),
	args ...any,
) (*Branch, error) {
	return tryConstruct(args, DefaultBranchConfig(), func(cfg BranchConfig) *Branch {
		return newBranch(condition, cfg)
	})
}

// When sets both the true and false paths for the branch.
func (b *Branch) When(truePath Flow) *Branch {
	b.truePath = truePath
	return b
}

// Else sets the false path for the branch.
func (b *Branch) Else(falsePath Flow) *Branch {
	b.falsePath = falsePath
	return b
}

// Run implements the Flow interface.
func (b *Branch) Run(ctx context.Context) error {
	// Call onStart hook if defined
	if b.onStart != nil {
		b.onStart(ctx, b.name, nil)
	}

	// Evaluate condition
	result, err := b.condition(ctx)
	if err != nil {
		if b.onError != nil {
			b.onError(ctx, b.name, err)
		}
		return NewFlowError(b.name, "condition", err)
	}

	// Execute appropriate path
	var executeErr error
	if result && b.truePath != nil {
		executeErr = b.truePath.Run(ctx)
	} else if !result && b.falsePath != nil {
		executeErr = b.falsePath.Run(ctx)
	}

	// Handle execution error
	if executeErr != nil {
		if b.onError != nil {
			b.onError(ctx, b.name, executeErr)
		}
		path := "true"
		if !result {
			path = "false"
		}
		return NewFlowError(b.name, fmt.Sprintf("%s_path", path), executeErr)
	}

	// Call onComplete hook if defined
	if b.onComplete != nil {
		b.onComplete(ctx, b.name, nil)
	}

	return nil
}

// BranchBuilder provides a fluent API for building conditional branches.
type BranchBuilder struct {
	condition func(ctx context.Context) (bool, error)
	config    BranchConfig
	truePath  Flow
	falsePath Flow
}

// NewBranchBuilder creates a new branch builder.
func NewBranchBuilder(condition func(ctx context.Context) (bool, error)) *BranchBuilder {
	return &BranchBuilder{
		condition: condition,
		config:    DefaultBranchConfig(),
	}
}

// Name sets the name of the branch.
func (bb *BranchBuilder) Name(name string) *BranchBuilder {
	bb.config.Name = name
	return bb
}

// When sets the flow to execute when the condition is true.
func (bb *BranchBuilder) When(truePath Flow) *BranchBuilder {
	bb.truePath = truePath
	return bb
}

// Else sets the flow to execute when the condition is false.
func (bb *BranchBuilder) Else(falsePath Flow) *BranchBuilder {
	bb.falsePath = falsePath
	return bb
}

// OnStart sets the start hook.
func (bb *BranchBuilder) OnStart(hook Hook) *BranchBuilder {
	bb.config.OnStart = hook
	return bb
}

// OnComplete sets the complete hook.
func (bb *BranchBuilder) OnComplete(hook Hook) *BranchBuilder {
	bb.config.OnComplete = hook
	return bb
}

// OnError sets the error hook.
func (bb *BranchBuilder) OnError(hook Hook) *BranchBuilder {
	bb.config.OnError = hook
	return bb
}

// Build returns the configured branch.
func (bb *BranchBuilder) Build() *Branch {
	branch := newBranch(bb.condition, bb.config)
	branch.truePath = bb.truePath
	branch.falsePath = bb.falsePath
	return branch
}

// Run builds and runs the branch.
func (bb *BranchBuilder) Run(ctx context.Context) error {
	return bb.Build().Run(ctx)
}
