package formfx

import (
	"fmt"

	"github.com/oxhq/tfx/runfx"
)

// Prompt is the low-level component. It is the central piece that is manipulated.
// It no longer contains a copy of the configuration.
type Prompt struct {
	NumOptions    int // The number of options available in the prompt.
	SelectedIndex int // Exported so custom KeyHandlers can manipulate it.
	keyHandler    KeyHandlerFunc

	Done     chan int
	Canceled chan struct{}
}

// NewPrompt creates a new Prompt with the given number of options and default index.
func NewPrompt(numOptions int, defaultIndex int) (*Prompt, error) {
	config := DefaultConfig()
	config.NumOptions = numOptions
	config.SelectedIndex = defaultIndex
	if err := config.sanitize(); err != nil {
		return nil, err
	}

	return &Prompt{
		NumOptions:    config.NumOptions,
		SelectedIndex: config.SelectedIndex,
		keyHandler:    config.keyHandler,
		Done:          make(chan int, 1),
		Canceled:      make(chan struct{}),
	}, nil
}

func DefaultConfig() PromptConfig {
	return PromptConfig{
		NumOptions:    2,                    // Default to Yes/No
		SelectedIndex: 0,                    // Default to Yes
		keyHandler:    HorizontalKeyHandler, // Default horizontal navigation
	}
}

// sanitizeConfig validates the PromptConfig to ensure it has valid values.
func (c *PromptConfig) sanitize() error {
	if c == nil {
		return fmt.Errorf("%w [PromptConfig]", ErrConfigNotSet)
	}

	if c.NumOptions < 1 {
		return fmt.Errorf("numOptions must be at least 1, got %d", c.NumOptions)
	}
	if c.SelectedIndex < 0 {
		c.SelectedIndex = 0 // Ensure non-negative index.
	}

	if c.SelectedIndex >= c.NumOptions {
		c.SelectedIndex = c.NumOptions - 1 // Clamp to the last option.
	}

	return nil
}

func HorizontalKeyHandler(p *Prompt, key runfx.Key) bool {
	if !key.IsSelector() {
		return false // No action for non-selector keys.
	}

	if key.IsCancel() {
		close(p.Canceled)
		return true // Stop on cancel.
	}
	if key.IsAccept() {
		if p.NumOptions > 0 {
			p.Done <- p.SelectedIndex // Send the selected index.
		}
		return true // Stop on accept.
	}

	switch key.Code {
	case runfx.KeyArrowLeft, runfx.KeyA:
		if p.SelectedIndex > 0 {
			p.SelectedIndex--
		}
	case runfx.KeyArrowRight, runfx.KeyD:
		if p.SelectedIndex < p.NumOptions-1 {
			p.SelectedIndex++
		}
	case runfx.KeyTab:
		// Cycle through options.
		p.SelectedIndex = (p.SelectedIndex + 1) % p.NumOptions
	}

	return false
}

// VerticalKeyHandler handles vertical navigation in a prompt.
func VerticalKeyHandler(p *Prompt, key runfx.Key) bool {
	if !key.IsSelector() {
		return false // No action for non-selector keys.
	}

	if key.IsCancel() {
		close(p.Canceled)
		return true // Stop on cancel.
	}
	if key.IsAccept() {
		if p.NumOptions > 0 {
			p.Done <- p.SelectedIndex // Send the selected index.
		}
		return true // Stop on accept.
	}

	switch key.Code {
	case runfx.KeyArrowUp, runfx.KeyW:
		if p.SelectedIndex > 0 {
			p.SelectedIndex--
		}
	case runfx.KeyArrowDown, runfx.KeyS:
		if p.SelectedIndex < p.NumOptions-1 {
			p.SelectedIndex++
		}
	case runfx.KeyTab:
		// Cycle through options.
		p.SelectedIndex = (p.SelectedIndex + 1) % p.NumOptions
	}

	return false
}

// SetKeyHandler allows setting a custom key handler for the prompt.
func (p *Prompt) SetKeyHandler(handler KeyHandlerFunc) {
	if handler == nil {
		return
	}
	p.keyHandler = handler
}

// OnKey implements the runfx.Interactive interface and delegates to the injected logic.
func (p *Prompt) OnKey(key runfx.Key) bool {
	return p.keyHandler(p, key)
}

// These methods are required by the runfx.Visual interface.
func (p *Prompt) Render() []byte          { return nil }
func (p *Prompt) OnResize(cols, rows int) {}
