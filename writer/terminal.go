// writer/terminal_writer.go
package writer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/terminal"
	"golang.org/x/term"
)

// TerminalOptions configures the terminal writer behavior.
type TerminalOptions struct {
	ForceColor   bool // force color support
	DisableColor bool // disable all colors
	DoubleBuffer bool // flicker-free updates
	TTYFile      *os.File
}

// TerminalWriter handles raw terminal output with double-buffering and color support.
type TerminalWriter struct {
	out      io.Writer
	detector *terminal.Detector

	mu      sync.Mutex
	prevBuf []byte

	opts TerminalOptions
}

// NewTerminalWriter creates a new TerminalWriter.
// Pass os.Stdout (or any *os.File) to support raw mode & size detection.
func NewTerminalWriter(out io.Writer, opts TerminalOptions) *TerminalWriter {
	detectorOutput := out
	if opts.TTYFile != nil {
		detectorOutput = opts.TTYFile
	}
	return &TerminalWriter{
		out:      out,
		detector: terminal.NewDetector(detectorOutput),
		prevBuf:  nil,
		opts:     opts,
	}
}

// Write implements io.Writer. Applies double-buffering if enabled.
func (w *TerminalWriter) Write(p []byte) (int, error) {
	if w.opts.DoubleBuffer {
		return w.writeBuffered(p)
	}
	return w.out.Write(p)
}

// writeBuffered writes only when content changes.
func (w *TerminalWriter) writeBuffered(cur []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if bytes.Equal(cur, w.prevBuf) {
		// simulate full write
		return len(cur), nil
	}
	// write full frame
	n, err := w.out.Write(cur)
	if err != nil {
		return n, err
	}
	// store copy of buffer
	w.prevBuf = append(w.prevBuf[:0], cur...)
	return n, nil
}

// SupportsColor reports whether ANSI is supported.
func (w *TerminalWriter) SupportsColor() bool {
	if w.opts.ForceColor {
		return true
	}
	if w.opts.DisableColor {
		return false
	}
	return w.detector.SupportsANSI()
}

// GetColorMode picks best available color mode.
func (w *TerminalWriter) GetColorMode() color.Mode {
	if w.opts.ForceColor {
		return color.ModeTrueColor
	}
	if !w.SupportsColor() {
		return color.ModeNoColor
	}
	mode := w.detector.GetMode()
	switch {
	case mode >= terminal.ModeTrueColor:
		return color.ModeTrueColor
	case mode >= terminal.Mode256:
		return color.Mode256Color
	case mode >= terminal.ModeANSI:
		return color.ModeANSI
	default:
		return color.ModeNoColor
	}
}

// IsTerminal reports if out is a terminal.
func (w *TerminalWriter) IsTerminal() bool {
	if file := w.opts.TTYFile; file != nil {
		return terminal.IsTerminal(file)
	}
	return terminal.IsTerminal(w.out)
}

// Clear erases the screen and resets cursor.
func (w *TerminalWriter) Clear() error {
	if !w.IsTerminal() {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.out.Write([]byte("\033[2J\033[H"))
	if err != nil {
		return err
	}
	w.prevBuf = w.prevBuf[:0]
	return nil
}

// MoveCursor positions cursor at 1-based row,col.
func (w *TerminalWriter) MoveCursor(row, col int) error {
	if !w.IsTerminal() {
		return nil
	}
	seq := fmt.Sprintf("\033[%d;%dH", row, col)
	_, err := w.out.Write([]byte(seq))
	return err
}

// HideCursor hides the terminal cursor.
func (w *TerminalWriter) HideCursor() error {
	if !w.IsTerminal() {
		return nil
	}
	_, err := w.out.Write([]byte("\033[?25l"))
	return err
}

// ShowCursor shows the terminal cursor.
func (w *TerminalWriter) ShowCursor() error {
	if !w.IsTerminal() {
		return nil
	}
	_, err := w.out.Write([]byte("\033[?25h"))
	return err
}

// GetSize returns terminal width and height.
func (w *TerminalWriter) GetSize() (cols, rows int, err error) {
	if file := w.opts.TTYFile; file != nil {
		return terminal.GetSizeForFD(file.Fd())
	}
	if file, ok := w.out.(*os.File); ok {
		return terminal.GetSizeForFD(file.Fd())
	}
	return terminal.GetSize()
}

// EnableRawMode puts terminal into raw mode.
func (w *TerminalWriter) EnableRawMode() (*term.State, error) {
	if file := w.opts.TTYFile; file != nil {
		return terminal.MakeRaw(file.Fd())
	}
	if f, ok := w.out.(*os.File); ok {
		return terminal.MakeRaw(f.Fd())
	}
	return nil, fmt.Errorf("terminal: raw mode not supported on this writer")
}

// RestoreMode resets terminal mode.
func (w *TerminalWriter) RestoreMode(state *term.State) error {
	if file := w.opts.TTYFile; file != nil {
		return terminal.RestoreTerminal(file.Fd(), state)
	}
	if f, ok := w.out.(*os.File); ok {
		return terminal.RestoreTerminal(f.Fd(), state)
	}
	return fmt.Errorf("terminal: restore mode not supported on this writer")
}

// Flush is a no-op (satisfies interface).
func (w *TerminalWriter) Flush() error { return nil }

// Close is a no-op (satisfies interface).
func (w *TerminalWriter) Close() error { return nil }
