package runfx

import (
	"io"
	"os"
	"time"
)

// Config provides structured configuration for RunFX Loop
type Config struct {
	TickInterval time.Duration
	Output       io.Writer
	Input        io.Reader
}

// DefaultConfig returns default configuration for RunFX
func DefaultConfig() Config {
	return Config{
		TickInterval: 50 * time.Millisecond, // Default 50ms for smooth animation
		Output:       os.Stdout,
		Input:        os.Stdin,
	}
}

// determineTickInterval returns the tick interval based on TTY and color mode
func determineTickInterval(tty TTYInfo) time.Duration {
	if !tty.IsTTY {
		return 250 * time.Millisecond
	}
	switch {
	case tty.TrueColor:
		return 16 * time.Millisecond
	case tty.ANSI:
		return 33 * time.Millisecond
	default:
		return 100 * time.Millisecond
	}
}
