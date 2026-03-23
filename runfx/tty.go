package runfx

import (
	"io"
	"os"

	"github.com/oxhq/tfx/writer"
)

// TTYInfo holds terminal capability details (uses writer.TerminalWriter for consistency)
type TTYInfo struct {
	IsTTY     bool
	TrueColor bool
	ANSI      bool
	NoColor   bool
}

// DetectTTY returns TTYInfo using the same detection logic as writer.TerminalWriter
func DetectTTY() TTYInfo {
	return DetectTTYForOutput(os.Stdout)
}

// DetectTTYForIO returns TTYInfo for the controlling terminal device selected
// from the provided output/input pair.
func DetectTTYForIO(output io.Writer, input io.Reader) TTYInfo {
	if ttyFile := resolveTTYFile(output, input); ttyFile != nil {
		return DetectTTYForOutput(ttyFile)
	}
	return DetectTTYForOutput(output)
}

// DetectTTYForOutput returns TTYInfo for a specific output writer
func DetectTTYForOutput(output io.Writer) TTYInfo {
	// Use TerminalWriter for consistent detection logic
	termWriter := writer.NewTerminalWriter(output, writer.TerminalOptions{})

	isTTY := termWriter.IsTerminal()
	supportsColor := termWriter.SupportsColor()
	colorMode := termWriter.GetColorMode()

	return TTYInfo{
		IsTTY:     isTTY,
		TrueColor: colorMode.String() == "TrueColor",
		ANSI:      supportsColor,
		NoColor:   !supportsColor,
	}
}

func resolveTTYFile(output io.Writer, input io.Reader) *os.File {
	if file, ok := output.(*os.File); ok {
		return file
	}
	if file, ok := input.(*os.File); ok {
		return file
	}
	return nil
}

// FallbackOutput prints minimal output if not TTY
func FallbackOutput(msg string) {
	if !DetectTTY().IsTTY {
		_, _ = os.Stdout.Write([]byte(msg + "\n"))
	}
}
