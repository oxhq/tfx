package formfx

import (
	"fmt"
	"strings"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
)

// --- 1. Configuration and Renderer ---

// SelectConfig contains the declarative configuration for a SelectPrompt.
type SelectConfig struct {
	Label         string
	Options       []string
	SelectedIndex int
	KeyHandler    KeyHandlerFunc
	Renderer      SelectRenderer
}

// DefaultSelectConfig returns the default configuration for a SelectPrompt.
func DefaultSelectConfig() SelectConfig {
	return SelectConfig{
		Label:         "Select an option:",
		SelectedIndex: 0,
		KeyHandler:    VerticalKeyHandler,
		Renderer:      &DefaultSelectRenderer{},
		Options:       []string{}, // The list of options must be provided by the user.
	}
}

// sanitize validates and corrects the configuration to ensure it is valid.
func (c *SelectConfig) sanitize() error {
	if len(c.Options) == 0 {
		return fmt.Errorf("options must not be empty")
	}
	if c.SelectedIndex < 0 {
		c.SelectedIndex = 0
	}
	if c.SelectedIndex >= len(c.Options) {
		c.SelectedIndex = len(c.Options) - 1
	}
	if c.KeyHandler == nil {
		c.KeyHandler = VerticalKeyHandler
	}
	if c.Renderer == nil {
		c.Renderer = &DefaultSelectRenderer{}
	}
	return nil
}

// SelectRenderer defines the interface for rendering a SelectPrompt component.
type SelectRenderer interface {
	Render(s *SelectPrompt) []byte
}

// DefaultSelectRenderer is the standard implementation.
type DefaultSelectRenderer struct{}

func (r *DefaultSelectRenderer) Render(s *SelectPrompt) []byte {
	var options []string
	for i, opt := range s.Options {
		if i == s.prompt.SelectedIndex {
			options = append(options, fmt.Sprintf("> %s", opt))
		} else {
			options = append(options, fmt.Sprintf("  %s", opt))
		}
	}
	return fmt.Appendf(nil, "%s\n%s\n", s.Label, strings.Join(options, "\n"))
}

// --- 2. The High-Level Component: SelectPrompt ---

// SelectPrompt is a UI component for selecting from a list of options.
type SelectPrompt struct {
	prompt   *Prompt
	Label    string
	Options  []string
	renderer SelectRenderer
}

// NewSelectPrompt is the explicit and strongly-typed constructor.
func NewSelectPrompt(cfg SelectConfig) (*SelectPrompt, error) {
	if err := cfg.sanitize(); err != nil {
		return nil, fmt.Errorf("invalid SelectConfig: %w", err)
	}

	p, err := NewPrompt(len(cfg.Options), cfg.SelectedIndex)
	if err != nil {
		return nil, err
	}
	p.SetKeyHandler(cfg.KeyHandler)

	return &SelectPrompt{
		Label:    cfg.Label,
		Options:  cfg.Options,
		prompt:   p,
		renderer: cfg.Renderer,
	}, nil
}

// --- 3. Convenience Functions (Multipath and DSL) ---

// Select is the high-level convenience function.
func Select(opts ...any) (*SelectPrompt, error) {
	cfg, err := share.TryOverloadWithOptions[SelectConfig](opts, DefaultSelectConfig())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidConfigType, err.Error())
	}
	return NewSelectPrompt(cfg)
}

// SelectBuilder provides the DSL path.
type SelectBuilder struct {
	config SelectConfig
}

// NewSelectBuilder is the entry point for the DSL path.
func NewSelectBuilder() *SelectBuilder {
	return &SelectBuilder{
		config: DefaultSelectConfig(),
	}
}

// Label sets the prompt label.
func (b *SelectBuilder) Label(label string) *SelectBuilder {
	b.config.Label = label
	return b
}

// Options sets the selection options.
func (b *SelectBuilder) Options(options []string) *SelectBuilder {
	b.config.Options = options
	return b
}

// SelectedIndex sets the default selection index.
func (b *SelectBuilder) SelectedIndex(index int) *SelectBuilder {
	b.config.SelectedIndex = index
	return b
}

// Build constructs the SelectPrompt with the provided configuration.
func (b *SelectBuilder) Build() (*SelectPrompt, error) {
	return NewSelectPrompt(b.config)
}

// --- Component Methods ---

func (s *SelectPrompt) SetRenderer(r SelectRenderer) {
	s.renderer = r
}

func (s *SelectPrompt) Done() <-chan int {
	return s.prompt.Done
}

func (s *SelectPrompt) Canceled() <-chan struct{} {
	return s.prompt.Canceled
}

// SetOptions allows dynamically changing the component's options.
// Recreates the internal prompt to maintain state consistency.
func (s *SelectPrompt) SetOptions(options []string) error {
	if len(options) == 0 {
		return fmt.Errorf("options cannot be empty")
	}
	s.Options = options
	oldPrompt := s.prompt
	handler := VerticalKeyHandler
	if oldPrompt != nil && oldPrompt.keyHandler != nil {
		handler = oldPrompt.keyHandler
	}

	// Ensures the selected index remains valid.
	newSelectedIndex := 0
	if oldPrompt != nil {
		newSelectedIndex = oldPrompt.SelectedIndex
	}
	if newSelectedIndex >= len(options) {
		newSelectedIndex = len(options) - 1
	}

	// REPLACEMENT: Creates a new, validated prompt. Does not mutate the previous one.
	newPrompt, err := NewPrompt(len(options), newSelectedIndex)
	if err != nil {
		return fmt.Errorf("failed to update internal prompt: %w", err)
	}
	newPrompt.SetKeyHandler(handler)
	if oldPrompt != nil {
		newPrompt.Done = oldPrompt.Done
		newPrompt.Canceled = oldPrompt.Canceled
	}
	s.prompt = newPrompt

	return nil
}

// --- RunFX Interface Implementation ---

func (s *SelectPrompt) Render() []byte {
	return s.renderer.Render(s)
}

func (s *SelectPrompt) OnKey(key runfx.Key) bool {
	return s.prompt.OnKey(key)
}

func (s *SelectPrompt) OnResize(cols, rows int) {}
