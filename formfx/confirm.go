package formfx

import (
	"fmt"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
)

// ConfirmPrompt is a high-level UI component for a binary choice (Yes/No).
// It uses a primitive Prompt internally to manage state.
type ConfirmPrompt struct {
	prompt   *Prompt
	Label    string
	renderer ConfirmRenderer // Renderer to customize visualization.
}

// ConfirmRenderer is the interface that defines how a ConfirmPrompt component should be rendered.
type ConfirmRenderer interface {
	Render(c *ConfirmPrompt) []byte
}

// DefaultConfirmRenderer is the standard implementation that draws "[Yes] / No".
type DefaultConfirmRenderer struct{}

// Render translates the state of ConfirmPrompt to a visual representation.
func (r *DefaultConfirmRenderer) Render(c *ConfirmPrompt) []byte {
	var yes, no string

	// Check the state of the internal prompt to decide how to draw.
	if c.prompt.SelectedIndex == 0 {
		yes = "[Yes]" // Style for the selected option.
		no = " No "
	} else {
		yes = " Yes "
		no = "[No]"
	}
	return fmt.Appendf(nil, "%s\n%s%s\n", c.Label, yes, no)
}

// ConfirmConfig holds the configuration for ConfirmPrompt.
type ConfirmConfig struct {
	Label        string          // The question to ask.
	DefaultValue bool            // Default selection (true for Yes, false for No).
	KeyHandler   KeyHandlerFunc  // Custom key handling logic.
	Renderer     ConfirmRenderer // Custom renderer for visualization.
}

// DefaultConfirmConfig returns a default configuration for ConfirmPrompt.
func DefaultConfirmConfig() *ConfirmConfig {
	return &ConfirmConfig{
		Label:        "Confirm your choice:",
		DefaultValue: true,                      // Default to Yes.
		KeyHandler:   HorizontalKeyHandler,      // Default horizontal navigation.
		Renderer:     &DefaultConfirmRenderer{}, // Default renderer.
	}
}

// sanitize validates the ConfirmConfig to ensure it has valid values.
func (c *ConfirmConfig) sanitize() error {
	if c == nil {
		return fmt.Errorf("%w [ConfirmConfig]", ErrConfigNotSet)
	}

	if c.Label == "" {
		return fmt.Errorf("label cannot be empty")
	}

	if c.KeyHandler == nil {
		c.KeyHandler = HorizontalKeyHandler // Default to horizontal navigation.
	}

	if c.Renderer == nil {
		c.Renderer = &DefaultConfirmRenderer{} // Use the default renderer.
	}
	return nil
}

// Confirm creates a new ConfirmPrompt with the given label and default selection.
// It defaults to "Yes" if defaultValue is true, otherwise "No".
func Confirm(args ...any) (*ConfirmPrompt, error) {
	cfg, err := share.TryOverloadWithOptions(args, DefaultConfirmConfig())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidConfigType, err.Error())
	}
	return NewConfirmPrompt(cfg)
}

// NewConfirmPrompt creates a new Confirm component.
// `label` is the question to ask.
// `defaultValue` sets the initial selection (0 for Yes, false for No).
// It panics if the defaultValue is not a boolean.
func NewConfirmPrompt(cfg *ConfirmConfig) (*ConfirmPrompt, error) {
	if err := cfg.sanitize(); err != nil {
		return nil, err
	}
	// Set the default index based on the defaultValue.
	defaultIndex := 0 // Default to Yes.
	if !cfg.DefaultValue {
		defaultIndex = 1 // Select No if defaultValue is false.
	}

	prompt, err := NewPrompt(2, defaultIndex)
	if err != nil {
		return nil, err
	}
	// Set the key handler and renderer.
	prompt.SetKeyHandler(cfg.KeyHandler)

	return &ConfirmPrompt{
		prompt:   prompt,
		Label:    cfg.Label,
		renderer: cfg.Renderer,
	}, nil
}

// SetRenderer allows changing the renderer of ConfirmPrompt.
func (c *ConfirmPrompt) SetRenderer(r ConfirmRenderer) {
	c.renderer = r
}

// Done returns a channel that will receive the result index when the user confirms.
// The caller is responsible for translating the index (0 for Yes, 1 for No).
func (c *ConfirmPrompt) Done() <-chan int {
	return c.prompt.Done
}

// Canceled returns a channel that will be closed if the user cancels the operation.
func (c *ConfirmPrompt) Canceled() <-chan struct{} {
	return c.prompt.Canceled
}

// Render implements the runfx.Visual interface.
// ConfirmPrompt is responsible for translating the state of the internal prompt
// to a "Yes/No" visual representation.
func (c *ConfirmPrompt) Render() []byte {
	return c.renderer.Render(c)
}

// OnKey implements the runfx.Interactive interface by delegating the call to the primitive prompt.
func (c *ConfirmPrompt) OnKey(key runfx.Key) bool {
	return c.prompt.OnKey(key)
}

// OnResize implements the runfx.Visual interface.
func (c *ConfirmPrompt) OnResize(cols, rows int) {}

type ConfirmBuilder struct {
	config *ConfirmConfig
	err    error
}

// NewConfirmBuilder is a non standard way to create a ConfirmPrompt.
// Its experimental and it's intended for builder patterns.
func NewConfirmBuilder(label string, defaultValue bool) *ConfirmBuilder {
	return &ConfirmBuilder{
		config: &ConfirmConfig{
			Label:        label,
			DefaultValue: defaultValue,
			KeyHandler:   HorizontalKeyHandler, // Default horizontal navigation.
			Renderer:     &DefaultConfirmRenderer{},
		},
	}
}

// KeyHandler sets a custom key handler for the ConfirmPrompt.
func (b *ConfirmBuilder) KeyHandler(handler KeyHandlerFunc) *ConfirmBuilder {
	if handler == nil {
		b.setError(ErrInvalidKeyHandler)
		return b
	}
	b.config.KeyHandler = handler
	return b
}

// Renderer sets a custom renderer for the ConfirmPrompt.
func (b *ConfirmBuilder) Renderer(renderer ConfirmRenderer) *ConfirmBuilder {
	if renderer == nil {
		b.setError(ErrInvalidRenderer)
		return b
	}
	b.config.Renderer = renderer
	return b
}

// DefaultValue sets the default selection for the ConfirmPrompt.
func (b *ConfirmBuilder) DefaultValue(defaultValue bool) *ConfirmBuilder {
	b.config.DefaultValue = defaultValue
	return b
}

// Label sets the label for the ConfirmPrompt.
func (b *ConfirmBuilder) Label(label string) *ConfirmBuilder {
	if label == "" {
		b.setError(fmt.Errorf("%w: label cannot be empty", ErrInvalidOption))
		return b
	}
	b.config.Label = label
	return b
}

// Build constructs the ConfirmPrompt with the provided configuration.
func (b *ConfirmBuilder) Build() (*ConfirmPrompt, error) {
	if b.err != nil {
		return nil, b.err
	}
	if err := b.config.sanitize(); err != nil {
		return nil, err
	}
	return NewConfirmPrompt(b.config)
}

func (b *ConfirmBuilder) setError(err error) {
	if b.err == nil {
		b.err = err
	}
}
