package formfx

import "github.com/oxhq/tfx/runfx"

// PromptConfig defines the configuration for the Prompt component.
type PromptConfig struct {
	NumOptions    int // The number of options available in the prompt.
	SelectedIndex int // Exported so custom KeyHandlers can manipulate it.
	keyHandler    KeyHandlerFunc
}

// KeyHandlerFunc defines the signature for injectable keyboard logic.
// Returns true if the loop should stop.
type KeyHandlerFunc func(p *Prompt, key runfx.Key) bool
