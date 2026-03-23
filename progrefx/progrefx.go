package progrefx

import (
	"fmt"
	"sync"
	"time"

	"github.com/oxhq/tfx/runfx"
	"github.com/oxhq/tfx/terminal"
)

// Progress represents a progress bar primitive.  It holds state about
// the total amount of work and how much has been completed.  A Progress
// can render itself as either a textual representation for non-TTY
// environments or an animated bar for TTY terminals.  Use the
// ProgressBuilder or Start functions to construct a new Progress.
type Progress struct {
	total     int
	current   int
	label     string
	startTime time.Time
	isStarted bool

	width    int
	theme    ProgressTheme
	style    ProgressStyle
	effect   ProgressEffect
	detector *terminal.Detector
	ShowETA  bool
	isTTY    bool

	mu sync.Mutex
}

// newProgress assembles a Progress from configuration.
func newProgress(cfg ProgressConfig) *Progress {
	detect := cfg.DetectTTY
	if detect == nil {
		detect = runfx.DetectTTY
	}
	theme := cfg.Theme
	if theme == (ProgressTheme{}) {
		theme = MaterialTheme
	}
	tty := detect()

	return &Progress{
		total:    cfg.Total,
		label:    cfg.Label,
		width:    cfg.Width,
		theme:    theme,
		style:    cfg.Style.normalized(),
		effect:   cfg.Effect,
		detector: terminal.NewDetector(cfg.Writer),
		ShowETA:  cfg.ShowETA,
		isTTY:    tty.IsTTY,
	}
}

// min returns the smaller of two integers.  Go doesn't provide a
// generic min for ints in the standard library, so we define our own here.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *Progress) effectiveStyle() ProgressStyle {
	return p.style.normalized()
}

// Render returns the current progress bar representation.  It falls back
// to plain text when not in a TTY.
func (p *Progress) Render() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isStarted {
		return ""
	}

	if !p.isTTY {
		percent := float64(p.current) / float64(p.total)
		return fmt.Sprintf("%s %3d%%", p.label, int(percent*100))
	}

	return RenderBar(p, p.detector)
}

// Set updates the progress to the given value.
func (p *Progress) Set(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isStarted {
		p.isStarted = true
		p.startTime = time.Now()
	}
	p.current = min(current, p.total)
}

// Add increments progress by the provided amount.
func (p *Progress) Add(amount int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isStarted {
		p.isStarted = true
		p.startTime = time.Now()
	}
	p.current = min(p.current+amount, p.total)
}

// SetLabel changes the progress label.
func (p *Progress) SetLabel(label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.label = label
}

// Finish sets the progress to 100%.
func (p *Progress) Finish() {
	p.Set(p.total)
}
