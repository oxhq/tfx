package formfx

import (
	"io"
	"os"

	"github.com/oxhq/tfx/terminal"
	"golang.org/x/term"
)

var oldTerminalState *term.State

// IsTTYFunc is a function type for checking if a writer is a TTY.
type IsTTYFunc func(w io.Writer) bool

// IsTTY is a variable that holds the current IsTTYFunc implementation.
var IsTTY IsTTYFunc = terminal.IsTerminal

// EnableEcho enables terminal echo.
func EnableEcho() error {
	return terminal.RestoreTerminal(os.Stdin.Fd(), oldTerminalState)
}

// DisableEcho disables terminal echo.
func DisableEcho() error {
	var err error
	oldTerminalState, err = terminal.MakeRaw(os.Stdin.Fd())
	return err
}
