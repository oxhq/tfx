package flowfx

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
)

// MapFlow represents a flow that executes steps based on key-value configuration.
// It allows dynamic step selection and configuration through maps.
type MapFlow struct {
	steps      map[string]Step
	order      []string // Execution order of steps
	name       string
	onStart    Hook
	onComplete Hook
	onError    Hook
	config     map[string]any // Configuration data
}

// MapFlowConfig provides configuration for a MapFlow.
type MapFlowConfig struct {
	Name       string
	OnStart    Hook
	OnComplete Hook
	OnError    Hook
	Config     map[string]any // Initial configuration
}

// DefaultMapFlowConfig returns the default configuration for a MapFlow.
func DefaultMapFlowConfig() MapFlowConfig {
	return MapFlowConfig{
		Name:   "mapflow",
		Config: make(map[string]any),
	}
}

// newMapFlow creates a new map flow with the given configuration.
func newMapFlow(cfg MapFlowConfig) *MapFlow {
	return &MapFlow{
		steps:      make(map[string]Step),
		order:      make([]string, 0),
		name:       cfg.Name,
		onStart:    cfg.OnStart,
		onComplete: cfg.OnComplete,
		onError:    cfg.OnError,
		config:     cloneMap(cfg.Config),
	}
}

// --- MULTIPATH API FUNCTIONS ---

// NewMapFlow creates a new map flow with multipath configuration support.
// Supports two usage patterns:
//   - NewMapFlow()                          // Zero-config, uses defaults
//   - NewMapFlow(config)                    // Config struct
//
// NewMapFlow is kept as a zero-ceremony compatibility alias for
// MustNewMapFlow. Prefer TryNewMapFlow when args may be user-provided or
// otherwise fallible.
func NewMapFlow(args ...any) *MapFlow {
	return MustNewMapFlow(args...)
}

// MustNewMapFlow creates a new map flow and panics on invalid multipath input.
func MustNewMapFlow(args ...any) *MapFlow {
	return mustValue(TryNewMapFlow(args...))
}

// TryNewMapFlow creates a new map flow without panicking on invalid args.
func TryNewMapFlow(args ...any) (*MapFlow, error) {
	return tryConstruct(args, DefaultMapFlowConfig(), newMapFlow)
}

// NewMapFlowBuilder creates a new MapFlowBuilder for DSL chaining.
func NewMapFlowBuilder() *MapFlowBuilder {
	return &MapFlowBuilder{config: DefaultMapFlowConfig()}
}

// AddStep adds a step with the given key to the flow.
func (mf *MapFlow) AddStep(key string, step Step) *MapFlow {
	if mf.steps == nil {
		mf.steps = make(map[string]Step)
	}
	mf.steps[key] = step
	mf.order = appendUnique(mf.order, key)
	return mf
}

// AddTask is a convenience method to add a Task as a step.
func (mf *MapFlow) AddTask(key string, task *Task) *MapFlow {
	return mf.AddStep(key, task)
}

// AddFunc is a convenience method to add a function as a step.
func (mf *MapFlow) AddFunc(key, label string, fn func(ctx context.Context) error) *MapFlow {
	task := NewTask(label, fn)
	return mf.AddStep(key, task)
}

// SetOrder sets the execution order of steps.
func (mf *MapFlow) SetOrder(order []string) *MapFlow {
	mf.order = dedupeStrings(cloneSlice(order))
	return mf
}

// SetConfig sets a configuration value.
func (mf *MapFlow) SetConfig(key string, value any) *MapFlow {
	mf.config[key] = value
	return mf
}

// GetConfig gets a configuration value.
func (mf *MapFlow) GetConfig(key string) (any, bool) {
	value, exists := mf.config[key]
	return value, exists
}

// Run executes steps according to the configured order.
// It implements the Flow interface.
func (mf *MapFlow) Run(ctx context.Context) error {
	if len(mf.steps) == 0 {
		return NewFlowError(mf.name, "", ErrEmptyFlow)
	}

	// Call onStart hook if provided
	if mf.onStart != nil {
		mf.onStart(ctx, mf.name, nil)
	}

	// Execute steps in order
	for _, key := range mf.order {
		select {
		case <-ctx.Done():
			err := NewFlowError(mf.name, key, ErrCanceled)
			if mf.onError != nil {
				mf.onError(ctx, mf.name, err)
			}
			return err
		default:
		}

		step, exists := mf.steps[key]
		if !exists {
			err := fmt.Errorf("step %q not found", key)
			flowErr := NewFlowError(mf.name, key, err)
			if mf.onError != nil {
				mf.onError(ctx, mf.name, flowErr)
			}
			return flowErr
		}

		// Execute the step
		if err := step.Execute(ctx); err != nil {
			flowErr := NewFlowError(mf.name, key, err)
			if mf.onError != nil {
				mf.onError(ctx, mf.name, flowErr)
			}
			return flowErr
		}
	}

	// All steps completed successfully
	if mf.onComplete != nil {
		mf.onComplete(ctx, mf.name, nil)
	}

	return nil
}

// Steps returns a copy of the steps map.
func (mf *MapFlow) Steps() map[string]Step {
	steps := make(map[string]Step)
	maps.Copy(steps, mf.steps)
	return steps
}

// Order returns a copy of the execution order.
func (mf *MapFlow) Order() []string {
	order := make([]string, len(mf.order))
	copy(order, mf.order)
	return order
}

// Config returns a copy of the configuration.
func (mf *MapFlow) Config() map[string]any {
	config := make(map[string]any)
	maps.Copy(config, mf.config)
	return config
}

// ToJSON serializes the flow configuration to JSON.
func (mf *MapFlow) ToJSON() ([]byte, error) {
	data := map[string]any{
		"name":   mf.name,
		"order":  mf.order,
		"config": mf.config,
	}
	return json.Marshal(data)
}

// FromJSON deserializes flow configuration from JSON.
// Note: This only restores configuration and order, not the actual steps.
func (mf *MapFlow) FromJSON(data []byte) error {
	var jsonData map[string]any
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if name, ok := jsonData["name"].(string); ok {
		mf.name = name
	}

	if orderData, ok := jsonData["order"].([]any); ok {
		mf.order = make([]string, len(orderData))
		for i, v := range orderData {
			if str, ok := v.(string); ok {
				mf.order[i] = str
			}
		}
	}

	if configData, ok := jsonData["config"].(map[string]any); ok {
		mf.config = make(map[string]any)
		maps.Copy(mf.config, configData)
	}

	return nil
}

// --- DSL BUILDER ---

// MapFlowBuilder provides a fluent API for building map flows.
type MapFlowBuilder struct {
	config MapFlowConfig
	steps  map[string]Step
	order  []string
}

// Name sets the name of the map flow.
func (mfb *MapFlowBuilder) Name(name string) *MapFlowBuilder {
	mfb.config.Name = name
	return mfb
}

// OnStart sets the start hook.
func (mfb *MapFlowBuilder) OnStart(hook Hook) *MapFlowBuilder {
	mfb.config.OnStart = hook
	return mfb
}

// OnComplete sets the complete hook.
func (mfb *MapFlowBuilder) OnComplete(hook Hook) *MapFlowBuilder {
	mfb.config.OnComplete = hook
	return mfb
}

// OnError sets the error hook.
func (mfb *MapFlowBuilder) OnError(hook Hook) *MapFlowBuilder {
	mfb.config.OnError = hook
	return mfb
}

// Config sets the initial configuration.
func (mfb *MapFlowBuilder) Config(config map[string]any) *MapFlowBuilder {
	mfb.config.Config = config
	return mfb
}

// Step adds a step to the map flow.
func (mfb *MapFlowBuilder) Step(key string, step Step) *MapFlowBuilder {
	if mfb.steps == nil {
		mfb.steps = make(map[string]Step)
	}
	mfb.steps[key] = step
	mfb.order = appendUnique(mfb.order, key)
	return mfb
}

// Task adds a task as a step to the map flow.
func (mfb *MapFlowBuilder) Task(key string, task *Task) *MapFlowBuilder {
	return mfb.Step(key, task)
}

// Func adds a function as a step to the map flow.
func (mfb *MapFlowBuilder) Func(
	key, label string,
	fn func(ctx context.Context) error,
) *MapFlowBuilder {
	task := NewTask(label, fn)
	return mfb.Step(key, task)
}

// Order sets the execution order of steps.
func (mfb *MapFlowBuilder) Order(order []string) *MapFlowBuilder {
	mfb.order = dedupeStrings(cloneSlice(order))
	return mfb
}

// Build creates a new MapFlow instance without running it.
func (mfb *MapFlowBuilder) Build() *MapFlow {
	mapFlow := newMapFlow(mfb.config)
	if mfb.steps != nil {
		mapFlow.steps = cloneMap(mfb.steps)
	}
	if mfb.order != nil {
		mapFlow.order = dedupeStrings(cloneSlice(mfb.order))
	}
	return mapFlow
}

// Run creates and runs the map flow.
func (mfb *MapFlowBuilder) Run(ctx context.Context) error {
	return mfb.Build().Run(ctx)
}
