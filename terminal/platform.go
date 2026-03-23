package terminal

import (
	"io"
	"os"

	"golang.org/x/term"
)

func fdToInt(fd uintptr) (int, error) {
	maxInt := ^uint(0) >> 1
	if fd > uintptr(maxInt) {
		return 0, os.ErrInvalid
	}
	// #nosec G115 -- the range check above guarantees the conversion is safe.
	return int(fd), nil
}

// IsTerminal detects if the file descriptor is a terminal
// This is the single entry point for all platform-specific terminal detection
func IsTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return isTerminal(v.Fd())
	default:
		return false
	}
}

// TryEnableANSI attempts to enable ANSI support on platforms that need it
// Returns true if ANSI support is available (either natively or successfully enabled)
func TryEnableANSI() bool {
	return enableANSI()
}

// MakeRaw puts the terminal into raw mode.
func MakeRaw(fd uintptr) (*term.State, error) {
	intFD, err := fdToInt(fd)
	if err != nil {
		return nil, err
	}
	return term.MakeRaw(intFD)
}

// RestoreTerminal restores the terminal to its original mode.
func RestoreTerminal(fd uintptr, state *term.State) error {
	intFD, err := fdToInt(fd)
	if err != nil {
		return err
	}
	return term.Restore(intFD, state)
}

// GetSize returns the terminal size (columns, rows)
func GetSize() (int, int, error) {
	return GetSizeForFD(os.Stdout.Fd())
}

// GetSizeForFD returns the terminal size for the given file descriptor.
func GetSizeForFD(fd uintptr) (int, int, error) {
	intFD, err := fdToInt(fd)
	if err != nil {
		return 0, 0, err
	}
	width, height, err := term.GetSize(intFD)
	return width, height, err
}

// GetSizeForFile returns the terminal size for the given file.
func GetSizeForFile(file *os.File) (int, int, error) {
	if file == nil {
		return 0, 0, os.ErrInvalid
	}
	return GetSizeForFD(file.Fd())
}
