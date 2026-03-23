package color

import (
	"math"
	"math/rand"
	"time"

	"github.com/oxhq/tfx/terminal"
)

// GradientString applies a linear gradient from the `from` color to the
// `to` color across the characters of the input string.  A detector
// should be supplied to choose the correct color mode for the
// terminal; if nil, ANSI mode is used.  Each character in the result
// will be wrapped in an escape sequence corresponding to its
// interpolated color.
func GradientString(text string, from, to Color, detector *terminal.Detector) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	var result string
	if len(runes) == 1 {
		mode := ModeANSI
		if detector != nil {
			mode = Mode(detector.GetMode())
		}
		return from.Render(mode) + string(runes[0]) + Reset
	}

	n := len(runes)
	for i, r := range runes {
		t := float64(i) / float64(n-1)
		rVal := uint8((1-t)*float64(from.R) + t*float64(to.R))
		gVal := uint8((1-t)*float64(from.G) + t*float64(to.G))
		bVal := uint8((1-t)*float64(from.B) + t*float64(to.B))
		c := NewRGB(rVal, gVal, bVal)
		mode := ModeANSI
		if detector != nil {
			mode = Mode(detector.GetMode())
		}
		result += c.Render(mode) + string(r) + Reset
	}
	return result
}

// RainbowString cycles through a rainbow palette for each character in
// the input string.  It returns a string with appropriate escape
// sequences.  The detector controls which color mode is used.
func RainbowString(text string, detector *terminal.Detector) string {
	if len(text) == 0 {
		return ""
	}
	rainbow := []Color{
		NewRGB(255, 0, 0),   // red
		NewRGB(255, 165, 0), // orange
		NewRGB(255, 255, 0), // yellow
		NewRGB(0, 128, 0),   // green
		NewRGB(0, 0, 255),   // blue
		NewRGB(75, 0, 130),  // indigo
		NewRGB(148, 0, 211), // violet
	}
	var result string
	mode := ModeANSI
	if detector != nil {
		mode = Mode(detector.GetMode())
	}
	for i, r := range []rune(text) {
		c := rainbow[i%len(rainbow)]
		result += c.Render(mode) + string(r) + Reset
	}
	return result
}

// WaveString applies a sine wave to the lightness of a base color
// across the characters of the input string.  The amplitude controls
// the magnitude of variation.  The result is a visually pleasing
// oscillation of brightness.
func WaveString(text string, base Color, amplitude uint8, detector *terminal.Detector) string {
	if len(text) == 0 {
		return ""
	}
	var result string
	mode := ModeANSI
	if detector != nil {
		mode = Mode(detector.GetMode())
	}
	for i, r := range []rune(text) {
		// Compute brightness modulation in the range [-1,1]
		factor := (math.Sin(float64(i)/2) + 1) / 2
		adjust := func(v uint8) uint8 {
			delta := float64(amplitude) * (factor - 0.5) * 2
			val := int(float64(v) + delta)
			if val < 0 {
				val = 0
			}
			if val > 255 {
				val = 255
			}
			return clampUint8(val)
		}
		c := NewRGB(adjust(base.R), adjust(base.G), adjust(base.B))
		result += c.Render(mode) + string(r) + Reset
	}
	return result
}

// GlitchString applies random colors to each character of the input
// string.  It uses a pseudo‑random generator seeded with the
// current time.  The result is visually chaotic and is intended for
// playful or experimental displays.
func GlitchString(text string, detector *terminal.Detector) string {
	if len(text) == 0 {
		return ""
	}
	// #nosec G404 -- this visual effect only needs lightweight pseudo-randomness.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var result string
	mode := ModeANSI
	if detector != nil {
		mode = Mode(detector.GetMode())
	}
	for _, r := range text {
		c := NewRGB(
			clampUint8(rng.Intn(256)),
			clampUint8(rng.Intn(256)),
			clampUint8(rng.Intn(256)),
		)
		result += c.Render(mode) + string(r) + Reset
	}
	return result
}
