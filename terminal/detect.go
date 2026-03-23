package terminal

import (
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
)

// Mode represents the capabilities of the output terminal
type Mode int

const (
	ModeNoColor   Mode = iota // No color support
	ModeANSI                  // ANSI color support
	Mode256                   // 256 color support
	ModeTrueColor             // True color (24-bit) support
)

// String returns a human-readable representation of the terminal mode
func (m Mode) String() string {
	switch m {
	case ModeNoColor:
		return "NoColor"
	case ModeANSI:
		return "ANSI"
	case Mode256:
		return "256Color"
	case ModeTrueColor:
		return "TrueColor"
	default:
		return "Unknown"
	}
}

// Detector handles terminal capability detection
type Detector struct {
	mode   Mode
	output io.Writer
	once   sync.Once
	mu     sync.RWMutex
}

// NewDetector creates a new terminal capability detector
func NewDetector(output io.Writer) *Detector {
	return &Detector{
		output: output,
	}
}

// GetMode returns the detected terminal mode
func (d *Detector) GetMode() Mode {
	d.once.Do(func() {
		d.mode = d.detectCapabilities()
	})
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.mode
}

// SetOutput changes the output writer and triggers re-detection
func (d *Detector) SetOutput(w io.Writer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.output = w
	d.once = sync.Once{} // Reset detection
}

// ForceMode manually sets the terminal mode
func (d *Detector) ForceMode(mode Mode) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.mode = mode
	d.once.Do(func() {}) // Mark as initialized
}

// SupportsColor returns true if the terminal supports any color
func (d *Detector) SupportsColor() bool {
	return d.GetMode() > ModeNoColor
}

// SupportsANSI returns true if the terminal supports ANSI escape codes
func (d *Detector) SupportsANSI() bool {
	return d.GetMode() >= ModeANSI
}

// Supports256Color returns true if the terminal supports 256 colors
func (d *Detector) Supports256Color() bool {
	return d.GetMode() >= Mode256
}

// SupportsTrueColor returns true if the terminal supports 24-bit color
func (d *Detector) SupportsTrueColor() bool {
	return d.GetMode() >= ModeTrueColor
}

// GetOS returns the current operating system
func (d *Detector) GetOS() string {
	return runtime.GOOS
}

// detectCapabilities performs the actual terminal capability detection
func (d *Detector) detectCapabilities() Mode {
	// Force no color if explicitly disabled
	if isColorDisabled() {
		return ModeNoColor
	}

	// Check if output is a terminal
	if !IsTerminal(d.output) {
		return ModeNoColor
	}

	// Check environment variables for color support
	colorTerm := os.Getenv("COLORTERM")
	term := os.Getenv("TERM")

	// True color support
	if colorTerm == "truecolor" || colorTerm == "24bit" {
		return ModeTrueColor
	}

	// 256 color support
	if strings.Contains(term, "256") || strings.Contains(term, "256color") {
		return Mode256
	}

	// Basic ANSI support
	if term != "" && term != "dumb" {
		// Common terminals that support ANSI
		ansiTerms := []string{
			"xterm", "screen", "tmux", "rxvt", "color", "ansi", "cygwin", "linux",
		}

		for _, ansiTerm := range ansiTerms {
			if strings.Contains(strings.ToLower(term), ansiTerm) {
				return ModeANSI
			}
		}
	}

	// Check for Windows terminal capabilities
	if runtime.GOOS == "windows" {
		if TryEnableANSI() {
			return ModeANSI
		}
	}

	return ModeNoColor
}

// isColorDisabled checks if color output is explicitly disabled
func isColorDisabled() bool {
	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return true
	}

	// Check for dumb terminal
	if os.Getenv("TERM") == "dumb" {
		return true
	}

	// Check if running in CI environment (usually no interactive terminal)
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "JENKINS_URL",
		"GITHUB_ACTIONS", "GITLAB_CI", "TRAVIS", "CIRCLECI",
	}

	for _, env := range ciVars {
		if os.Getenv(env) != "" {
			// Some CI systems support color, check FORCE_COLOR
			if os.Getenv("FORCE_COLOR") != "" {
				return false
			}
			return true
		}
	}

	return false
}
