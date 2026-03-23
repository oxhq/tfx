package progrefx

import (
	"fmt"
	"sync"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/runfx"
	"github.com/oxhq/tfx/terminal"
)

// Spinner is a simple animated indicator that cycles through frames.
// Spinners can be rendered in any environment and will gracefully fall
// back to plain text when no TTY is detected.
type Spinner struct {
	frames   []string
	index    int
	label    string
	theme    ProgressTheme
	detector *terminal.Detector
	isTTY    bool

	mu sync.Mutex
}

// newSpinner creates a Spinner using the provided configuration.
func newSpinner(cfg SpinnerConfig) *Spinner {
	detect := cfg.DetectTTY
	if detect == nil {
		detect = runfx.DetectTTY
	}
	tty := detect()

	return &Spinner{
		frames:   cfg.Frames,
		label:    cfg.Label,
		theme:    cfg.Theme,
		detector: terminal.NewDetector(cfg.Writer),
		isTTY:    tty.IsTTY,
	}
}

// Render returns the current spinner frame with the label.
// When not running in a TTY, only the label is returned.
func (s *Spinner) Render() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isTTY {
		return s.label
	}

	frame := s.frames[s.index%len(s.frames)]
	frameColor := s.theme.RenderColor(s.theme.CompleteColor, s.detector)
	labelColor := s.theme.RenderColor(s.theme.LabelColor, s.detector)

	styledFrame := frameColor + frame + color.Reset
	styledLabel := labelColor + s.label + color.Reset

	return fmt.Sprintf("\r%s %s", styledFrame, styledLabel)
}

// Tick advances the spinner to the next frame.
func (s *Spinner) Tick() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index = (s.index + 1) % len(s.frames)
}

// SetLabel updates the spinner's label text.
func (s *Spinner) SetLabel(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.label = label
}
