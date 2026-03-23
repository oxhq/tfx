package color

import "github.com/oxhq/tfx/internal/share"

// Re-export commonly used ANSI sequences for backward compatibility.
var (
	Reset     = share.Reset
	Bold      = share.Bold
	Dim       = share.Dim
	Italic    = share.Italic
	Underline = share.Underline
	Blink     = share.Blink
	Reverse   = share.Reverse
	Strike    = share.Strike
)

// ANSISeq provides access to raw ANSI escape sequences.
var ANSISeq = share.ANSISeq
