package progrefx

import (
	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/terminal"
)

// ProgressTheme defines the set of colors and behaviors used when
// rendering a progress bar or spinner. Themes can enable special
// effects (gradients, rainbows) that are rendered dynamically.
type ProgressTheme struct {
	Name            string
	CompleteColor   color.Color
	IncompleteColor color.Color
	LabelColor      color.Color
	PercentColor    color.Color
	BorderColor     color.Color // Frames/brackets color
	EffectEnabled   bool        // Enable gradient/rainbow effects
}

// NewThemeFromPalette creates a theme from a named palette.  It
// attempts to select sensible defaults based on common color names in
// the palette.  If a color is not present, a fallback is used.
func NewThemeFromPalette(name string, palette color.Palette) ProgressTheme {
	complete, _ := palette.Get("green")
	if complete == (color.Color{}) {
		complete = color.MaterialGreen
	}

	incomplete, _ := palette.Get("gray")
	if incomplete == (color.Color{}) {
		incomplete = color.NewANSI(8)
	}

	label, _ := palette.Get("blue")
	if label == (color.Color{}) {
		label = color.MaterialBlue
	}

	return ProgressTheme{
		Name:            name,
		CompleteColor:   complete,
		IncompleteColor: incomplete,
		LabelColor:      label,
		PercentColor:    label,
		BorderColor:     color.NewANSI(7),
		EffectEnabled:   false,
	}
}

// Professional and fun themes inspired by popular color schemes.  These
// variables define the built-in progrefx themes. You can choose any of
// these for your progress bars or extend them.
var (
	MaterialTheme = ProgressTheme{
		Name:            "material",
		CompleteColor:   color.MaterialGreen,
		IncompleteColor: color.NewANSI(8),
		LabelColor:      color.MaterialBlue,
		PercentColor:    color.MaterialCyan,
		BorderColor:     color.NewANSI(7),
		EffectEnabled:   false,
	}

	DraculaTheme = ProgressTheme{
		Name:            "dracula",
		CompleteColor:   color.DraculaGreen,
		IncompleteColor: color.RGB(68, 71, 90),
		LabelColor:      color.DraculaPurple,
		PercentColor:    color.DraculaCyan,
		BorderColor:     color.DraculaPink,
		EffectEnabled:   true,
	}

	NordTheme = ProgressTheme{
		Name:            "nord",
		CompleteColor:   color.NordGreen,
		IncompleteColor: color.RGB(59, 66, 82),
		LabelColor:      color.NordBlue,
		PercentColor:    color.NordCyan,
		BorderColor:     color.NordLightBlue,
		EffectEnabled:   false,
	}

	GitHubTheme = ProgressTheme{
		Name:            "github",
		CompleteColor:   color.GithubGreenLight,
		IncompleteColor: color.NewANSI(8),
		LabelColor:      color.GithubBlueLight,
		PercentColor:    color.GithubBlueLight,
		BorderColor:     color.NewANSI(7),
		EffectEnabled:   false,
	}

	TailwindTheme = ProgressTheme{
		Name:            "tailwind",
		CompleteColor:   color.TailwindGreen,
		IncompleteColor: color.NewANSI(8),
		LabelColor:      color.TailwindBlue,
		PercentColor:    color.TailwindCyan,
		BorderColor:     color.TailwindIndigo,
		EffectEnabled:   false,
	}

	VSCodeTheme = ProgressTheme{
		Name:            "vscode",
		CompleteColor:   color.VSCodeGreen,
		IncompleteColor: color.NewANSI(8),
		LabelColor:      color.VSCodeBlue,
		PercentColor:    color.VSCodeCyan,
		BorderColor:     color.NewANSI(7),
		EffectEnabled:   false,
	}

	RainbowTheme = ProgressTheme{
		Name:            "rainbow",
		CompleteColor:   color.MaterialGreen,
		IncompleteColor: color.NewANSI(8),
		LabelColor:      color.MaterialPurple,
		PercentColor:    color.MaterialCyan,
		BorderColor:     color.NewANSI(7),
		EffectEnabled:   true,
	}

	NeonTheme = ProgressTheme{
		Name:            "neon",
		CompleteColor:   color.RGB(0, 255, 0),
		IncompleteColor: color.NewANSI(8),
		LabelColor:      color.RGB(0, 150, 255),
		PercentColor:    color.RGB(255, 0, 255),
		BorderColor:     color.RGB(255, 255, 0),
		EffectEnabled:   true,
	}
)

// AllProgressThemes groups all defined progress themes for easy iteration.
var AllProgressThemes = []ProgressTheme{
	MaterialTheme,
	DraculaTheme,
	NordTheme,
	GitHubTheme,
	TailwindTheme,
	VSCodeTheme,
	RainbowTheme,
	NeonTheme,
}

// RenderColor returns an ANSI escape sequence for a given color.  If a
// detector is provided, it will choose the best color mode supported by
// the terminal.  Otherwise it falls back to ANSI mode.
func (pt ProgressTheme) RenderColor(c color.Color, detector *terminal.Detector) string {
	if detector == nil {
		return c.Render(color.ModeANSI)
	}
	terminalMode := color.Mode(detector.GetMode())
	return c.Render(terminalMode)
}

// ProgressEffect enumerates supported visual effects for progress bars.
type ProgressEffect int

const (
	EffectNone ProgressEffect = iota
	EffectGradient
	EffectRainbow
	EffectPulse
	EffectGlow
)

// RenderProgress renders a progress bar using the requested effect.  The
// filled parameter is the number of characters that should be drawn.
func (pt ProgressTheme) RenderProgress(
	percent float64,
	width int,
	effect ProgressEffect,
	style ProgressStyle,
	detector *terminal.Detector,
) string {
	filled := int(percent * float64(width))
	switch effect {
	case EffectRainbow:
		return pt.renderRainbowProgress(filled, width, style, detector)
	case EffectGradient:
		return pt.renderGradientProgress(filled, width, style, detector)
	case EffectGlow:
		return pt.renderGlowProgress(filled, width, style, detector)
	default:
		return pt.renderSolidProgress(filled, width, style, detector)
	}
}

// renderSolidProgress renders a progress bar without any special effects.
func (pt ProgressTheme) renderSolidProgress(
	filled, width int,
	style ProgressStyle,
	detector *terminal.Detector,
) string {
	completeColor := pt.RenderColor(pt.CompleteColor, detector)
	incompleteColor := pt.RenderColor(pt.IncompleteColor, detector)
	bar := ""
	for i := range width {
		if i < filled {
			bar += completeColor + style.FilledChar() + color.Reset
		} else {
			bar += incompleteColor + style.EmptyChar() + color.Reset
		}
	}
	return bar
}

// Gradient effect: smoothly transitions from incomplete to complete color across the bar.
func (pt ProgressTheme) renderGradientProgress(
	filled, width int,
	style ProgressStyle,
	detector *terminal.Detector,
) string {
	bar := ""
	for i := 0; i < width; i++ {
		// Interpolate between incomplete and complete colors based on position
		ratio := float64(i) / float64(width-1)
		r := uint8(float64(pt.IncompleteColor.R)*(1-ratio) + float64(pt.CompleteColor.R)*ratio)
		g := uint8(float64(pt.IncompleteColor.G)*(1-ratio) + float64(pt.CompleteColor.G)*ratio)
		b := uint8(float64(pt.IncompleteColor.B)*(1-ratio) + float64(pt.CompleteColor.B)*ratio)
		c := color.NewRGB(r, g, b)
		mode := color.Mode(detector.GetMode())
		if i < filled {
			bar += c.Render(mode) + style.FilledChar() + color.Reset
		} else {
			bar += c.Render(mode) + style.EmptyChar() + color.Reset
		}
	}
	return bar
}

// Rainbow effect: cycles through a set of colors to create a rainbow.
func (pt ProgressTheme) renderRainbowProgress(
	filled, width int,
	style ProgressStyle,
	detector *terminal.Detector,
) string {
	rainbow := []color.Color{
		color.NewRGB(255, 0, 0),   // red
		color.NewRGB(255, 165, 0), // orange
		color.NewRGB(255, 255, 0), // yellow
		color.NewRGB(0, 128, 0),   // green
		color.NewRGB(0, 0, 255),   // blue
		color.NewRGB(75, 0, 130),  // indigo
		color.NewRGB(148, 0, 211), // violet
	}
	bar := ""
	for i := 0; i < width; i++ {
		c := rainbow[i%len(rainbow)]
		mode := color.Mode(detector.GetMode())
		if i < filled {
			bar += c.Render(mode) + style.FilledChar() + color.Reset
		} else {
			bar += c.Render(mode) + style.EmptyChar() + color.Reset
		}
	}
	return bar
}

// Glow effect: emphasises the completed portion by brightening the complete color.
func (pt ProgressTheme) renderGlowProgress(
	filled, width int,
	style ProgressStyle,
	detector *terminal.Detector,
) string {
	bar := ""
	for i := 0; i < width; i++ {
		var c color.Color
		if i < filled {
			// Increase brightness of the complete color
			r := minInt(int(pt.CompleteColor.R)*2, 255)
			g := minInt(int(pt.CompleteColor.G)*2, 255)
			b := minInt(int(pt.CompleteColor.B)*2, 255)
			c = color.NewRGB(intToUint8(r), intToUint8(g), intToUint8(b))
		} else {
			c = pt.IncompleteColor
		}
		mode := color.Mode(detector.GetMode())
		if i < filled {
			bar += c.Render(mode) + style.FilledChar() + color.Reset
		} else {
			bar += c.Render(mode) + style.EmptyChar() + color.Reset
		}
	}
	return bar
}

// minInt is a helper for the glow effect.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intToUint8(value int) uint8 {
	switch {
	case value <= 0:
		return 0
	case value >= 255:
		return 255
	default:
		return uint8(value)
	}
}
