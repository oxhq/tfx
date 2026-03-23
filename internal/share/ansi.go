package share

// ANSI Escape Codes and Control Sequences
// These are the raw ANSI escape sequences for direct terminal control

const (
	// Control sequences
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	// Text styles
	Italic    = "\033[3m"
	Underline = "\033[4m"
	Blink     = "\033[5m"
	Reverse   = "\033[7m"
	Strike    = "\033[9m"
)

// ANSIColors provides direct access to ANSI color escape sequences
type ANSIColors struct{}

// ANSISeq provides all ANSI escape sequences
var ANSISeq = &ANSIColors{}

// Foreground colors
func (a *ANSIColors) Black() string   { return "\033[30m" }
func (a *ANSIColors) Red() string     { return "\033[31m" }
func (a *ANSIColors) Green() string   { return "\033[32m" }
func (a *ANSIColors) Yellow() string  { return "\033[33m" }
func (a *ANSIColors) Blue() string    { return "\033[34m" }
func (a *ANSIColors) Magenta() string { return "\033[35m" }
func (a *ANSIColors) Cyan() string    { return "\033[36m" }
func (a *ANSIColors) White() string   { return "\033[37m" }

// Bright foreground colors
func (a *ANSIColors) BrightBlack() string   { return "\033[90m" }
func (a *ANSIColors) BrightRed() string     { return "\033[91m" }
func (a *ANSIColors) BrightGreen() string   { return "\033[92m" }
func (a *ANSIColors) BrightYellow() string  { return "\033[93m" }
func (a *ANSIColors) BrightBlue() string    { return "\033[94m" }
func (a *ANSIColors) BrightMagenta() string { return "\033[95m" }
func (a *ANSIColors) BrightCyan() string    { return "\033[96m" }
func (a *ANSIColors) BrightWhite() string   { return "\033[97m" }

// Background colors
func (a *ANSIColors) BgBlack() string   { return "\033[40m" }
func (a *ANSIColors) BgRed() string     { return "\033[41m" }
func (a *ANSIColors) BgGreen() string   { return "\033[42m" }
func (a *ANSIColors) BgYellow() string  { return "\033[43m" }
func (a *ANSIColors) BgBlue() string    { return "\033[44m" }
func (a *ANSIColors) BgMagenta() string { return "\033[45m" }
func (a *ANSIColors) BgCyan() string    { return "\033[46m" }
func (a *ANSIColors) BgWhite() string   { return "\033[47m" }

// Bright background colors
func (a *ANSIColors) BgBrightBlack() string   { return "\033[100m" }
func (a *ANSIColors) BgBrightRed() string     { return "\033[101m" }
func (a *ANSIColors) BgBrightGreen() string   { return "\033[102m" }
func (a *ANSIColors) BgBrightYellow() string  { return "\033[103m" }
func (a *ANSIColors) BgBrightBlue() string    { return "\033[104m" }
func (a *ANSIColors) BgBrightMagenta() string { return "\033[105m" }
func (a *ANSIColors) BgBrightCyan() string    { return "\033[106m" }
func (a *ANSIColors) BgBrightWhite() string   { return "\033[107m" }

// Convenience functions for common operations
func (a *ANSIColors) Reset() string { return Reset }
func (a *ANSIColors) Bold() string  { return Bold }
func (a *ANSIColors) Dim() string   { return Dim }

// Style text with ANSI sequences
func (a *ANSIColors) Style(text, colorSeq string) string {
	return colorSeq + text + Reset
}

// Wrap text with foreground and background colors
func (a *ANSIColors) Wrap(text, fg, bg string) string {
	return fg + bg + text + Reset
}
