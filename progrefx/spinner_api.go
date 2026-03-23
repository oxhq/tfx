package progrefx

import (
	"io"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
)

// SpinnerConfig holds configuration for Spinner.  Note that Theme now
// refers to ProgressTheme from this package; spinners and progress bars
// share themes for consistent styling.
type SpinnerConfig struct {
	Label     string
	Frames    []string
	Theme     ProgressTheme
	Writer    io.Writer // Used only for TTY detection.
	DetectTTY func() runfx.TTYInfo
}

// DefaultSpinnerConfig provides sensible defaults for a spinner.
func DefaultSpinnerConfig() SpinnerConfig {
	return SpinnerConfig{
		Label:     "Loading",
		Frames:    []string{"|", "/", "-", "\\"},
		Theme:     MaterialTheme,
		DetectTTY: runfx.DetectTTY,
	}
}

// StartSpinner is a convenience function that creates a Spinner using a
// configuration or a sequence of functional options.  See
// internal/share.OverloadWithOptions for details.
func StartSpinner(opts ...any) *Spinner {
	spinner, err := TryStartSpinner(opts...)
	if err != nil {
		panic(err)
	}
	return spinner
}

// TryStartSpinner creates a Spinner using the provided options without panicking.
func TryStartSpinner(opts ...any) (*Spinner, error) {
	cfg, err := share.TryOverloadWithOptions[SpinnerConfig](opts, DefaultSpinnerConfig())
	if err != nil {
		return nil, err
	}
	return newSpinner(cfg), nil
}

// SpinnerBuilder offers a fluent DSL for building a Spinner.
type SpinnerBuilder struct {
	config SpinnerConfig
}

// NewSpinnerBuilder creates a new builder with default configuration.
func NewSpinnerBuilder() *SpinnerBuilder {
	return &SpinnerBuilder{config: DefaultSpinnerConfig()}
}

// Label sets the spinner label.
func (b *SpinnerBuilder) Label(label string) *SpinnerBuilder {
	b.config.Label = label
	return b
}

// Frames sets custom spinner frames.
func (b *SpinnerBuilder) Frames(frames []string) *SpinnerBuilder {
	b.config.Frames = frames
	return b
}

// Theme sets the spinner theme.
func (b *SpinnerBuilder) Theme(theme ProgressTheme) *SpinnerBuilder {
	b.config.Theme = theme
	return b
}

// Writer sets the writer used for TTY detection.
func (b *SpinnerBuilder) Writer(writer io.Writer) *SpinnerBuilder {
	b.config.Writer = writer
	return b
}

// DetectTTY allows providing a custom TTY detection function.
func (b *SpinnerBuilder) DetectTTY(fn func() runfx.TTYInfo) *SpinnerBuilder {
	b.config.DetectTTY = fn
	return b
}

// Build constructs the Spinner with the configured options.
func (b *SpinnerBuilder) Build() *Spinner {
	return newSpinner(b.config)
}
