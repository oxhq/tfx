package progrefx

import (
	"io"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
)

// StepperConfig holds configuration for a stepper view.
type StepperConfig struct {
	Label     string
	Steps     []StepItem
	Theme     ProgressTheme
	Writer    io.Writer
	DetectTTY func() runfx.TTYInfo
}

// DefaultStepperConfig returns sensible defaults for a stepper view.
func DefaultStepperConfig() StepperConfig {
	return StepperConfig{
		Label:     "Steps",
		Theme:     MaterialTheme,
		DetectTTY: runfx.DetectTTY,
	}
}

// Stepper creates a stepper view using multipath configuration.
func Stepper(opts ...any) *StepperView {
	stepper, err := TryStepper(opts...)
	if err != nil {
		panic(err)
	}
	return stepper
}

// TryStepper creates a stepper view without panicking on invalid args.
func TryStepper(opts ...any) (*StepperView, error) {
	cfg, err := share.TryOverloadWithOptions(opts, DefaultStepperConfig())
	if err != nil {
		return nil, err
	}
	return newStepper(cfg), nil
}

// StepperBuilder provides a fluent builder API for stepper construction.
type StepperBuilder struct {
	config StepperConfig
}

// NewStepperBuilder returns a builder with default stepper configuration.
func NewStepperBuilder() *StepperBuilder {
	return &StepperBuilder{config: DefaultStepperConfig()}
}

// Label sets the stepper label.
func (b *StepperBuilder) Label(label string) *StepperBuilder {
	b.config.Label = label
	return b
}

// Steps sets the full step list.
func (b *StepperBuilder) Steps(steps []StepItem) *StepperBuilder {
	b.config.Steps = cloneSteps(steps)
	return b
}

// Theme sets the stepper theme.
func (b *StepperBuilder) Theme(theme ProgressTheme) *StepperBuilder {
	b.config.Theme = theme
	return b
}

// Writer sets the writer used for TTY detection.
func (b *StepperBuilder) Writer(writer io.Writer) *StepperBuilder {
	b.config.Writer = writer
	return b
}

// DetectTTY allows providing a custom TTY detection function.
func (b *StepperBuilder) DetectTTY(fn func() runfx.TTYInfo) *StepperBuilder {
	b.config.DetectTTY = fn
	return b
}

// Build constructs the configured stepper.
func (b *StepperBuilder) Build() *StepperView {
	return newStepper(b.config)
}
