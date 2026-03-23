package progrefx

import (
	"io"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
)

// ProgressConfig defines options for a progress bar.
// This lives under progrefx as the canonical progress API. Progress
// components created via this config are compatible with runfx for
// interactive rendering.
type ProgressConfig struct {
	Total     int
	Label     string
	Width     int
	Theme     ProgressTheme
	Style     ProgressStyle
	Effect    ProgressEffect
	Writer    io.Writer // Used only for TTY detection, not direct writes.
	ShowETA   bool
	DetectTTY func() runfx.TTYInfo
}

// DefaultProgressConfig returns sensible defaults for a progress bar.
func DefaultProgressConfig() ProgressConfig {
	return ProgressConfig{
		Total:     100,
		Label:     "Progress",
		Width:     40,
		Theme:     MaterialTheme,
		Style:     ProgressStyleBar,
		DetectTTY: runfx.DetectTTY,
	}
}

// Start creates a Progress component using the provided options.  It accepts
// either a ProgressConfig or a sequence of functional options.  See
// internal/share.OverloadWithOptions for details.
func Start(opts ...any) *Progress {
	progress, err := TryStart(opts...)
	if err != nil {
		panic(err)
	}
	return progress
}

// TryStart creates a Progress component using the provided options without panicking.
func TryStart(opts ...any) (*Progress, error) {
	cfg, err := share.TryOverloadWithOptions[ProgressConfig](opts, DefaultProgressConfig())
	if err != nil {
		return nil, err
	}
	return newProgress(cfg), nil
}

// ProgressBuilder provides a fluent builder API for constructing a progress bar.
// It is useful when you need to set only a few fields on the ProgressConfig.
type ProgressBuilder struct {
	config ProgressConfig
}

// NewProgressBuilder returns a builder with default configuration.
func NewProgressBuilder() *ProgressBuilder {
	return &ProgressBuilder{config: DefaultProgressConfig()}
}

// Total sets the total amount of work.
func (b *ProgressBuilder) Total(total int) *ProgressBuilder {
	b.config.Total = total
	return b
}

// Label sets the progress label.
func (b *ProgressBuilder) Label(label string) *ProgressBuilder {
	b.config.Label = label
	return b
}

// Width sets the bar width.
func (b *ProgressBuilder) Width(width int) *ProgressBuilder {
	b.config.Width = width
	return b
}

// Theme sets the progress theme.
func (b *ProgressBuilder) Theme(theme ProgressTheme) *ProgressBuilder {
	b.config.Theme = theme
	return b
}

// Style sets the progress style.
func (b *ProgressBuilder) Style(style ProgressStyle) *ProgressBuilder {
	b.config.Style = style
	return b
}

// Effect sets the progress effect.
func (b *ProgressBuilder) Effect(effect ProgressEffect) *ProgressBuilder {
	b.config.Effect = effect
	return b
}

// ShowETA toggles ETA rendering.
func (b *ProgressBuilder) ShowETA(show bool) *ProgressBuilder {
	b.config.ShowETA = show
	return b
}

// Writer sets the writer used for TTY detection.
func (b *ProgressBuilder) Writer(writer io.Writer) *ProgressBuilder {
	b.config.Writer = writer
	return b
}

// DetectTTY allows providing a custom TTY detection function.
func (b *ProgressBuilder) DetectTTY(fn func() runfx.TTYInfo) *ProgressBuilder {
	b.config.DetectTTY = fn
	return b
}

// Build constructs the Progress component with the configured options.
func (b *ProgressBuilder) Build() *Progress {
	return newProgress(b.config)
}
