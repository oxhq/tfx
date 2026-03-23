package progrefx

// ProgressStyle defines the shape used to render a progress bar.  It
// determines which characters are used for the filled and empty
// portions of the bar.  These styles mirror those available in
// progrefx package.
type ProgressStyle int

const (
	ProgressStyleBar ProgressStyle = iota
	ProgressStyleDots
	ProgressStyleArrows
	ProgressStyleAscii
)

// styleChars maps each style to its filled and empty symbols.
var styleChars = map[ProgressStyle]struct {
	Filled, Empty string
}{
	ProgressStyleBar:    {"█", "░"},
	ProgressStyleDots:   {"●", "○"},
	ProgressStyleArrows: {">", "-"},
	ProgressStyleAscii:  {"=", "-"},
}

func (s ProgressStyle) normalized() ProgressStyle {
	if _, ok := styleChars[s]; !ok {
		return ProgressStyleBar
	}
	return s
}

// FilledChar returns the rune used to represent completed work for this style.
func (s ProgressStyle) FilledChar() string { return styleChars[s.normalized()].Filled }

// EmptyChar returns the rune used to represent remaining work for this style.
func (s ProgressStyle) EmptyChar() string { return styleChars[s.normalized()].Empty }
