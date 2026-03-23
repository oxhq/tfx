// Package color provides a multipath API for terminal colors with ANSI, 256-color, and TrueColor support.
//
// Canonical color and theme data lives in palettes.go. system.go keeps the
// legacy global theme surface for compatibility with older callers and with
// packages that still depend on it.
package color

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/oxhq/tfx/internal/share"
)

// Color represents a color that can be rendered in different terminal modes.
type Color struct {
	R, G, B  uint8
	ANSI     int
	Color256 int
	Hex      string
	Name     string
	IsBg     bool
}

// ColorConfig provides structured configuration for Color creation.
type ColorConfig struct {
	R, G, B  uint8
	ANSI     int
	Color256 int
	Hex      string
	Name     string
	IsBg     bool
	Mode     Mode
}

// Mode represents different color rendering modes.
type Mode int

const (
	ModeNoColor Mode = iota
	ModeANSI
	Mode256Color
	ModeTrueColor
)

// String returns the string representation of a Mode.
func (m Mode) String() string {
	switch m {
	case ModeNoColor:
		return "NoColor"
	case ModeANSI:
		return "ANSI"
	case Mode256Color:
		return "256Color"
	case ModeTrueColor:
		return "TrueColor"
	default:
		return "Unknown"
	}
}

// DefaultColorConfig returns the default configuration for Color.
func DefaultColorConfig() ColorConfig {
	return ColorConfig{
		R:    128,
		G:    128,
		B:    128,
		Mode: ModeTrueColor,
		IsBg: false,
	}
}

func NewRGB(r, g, b uint8) Color {
	return NewColor(ColorConfig{R: r, G: g, B: b})
}

func NewHex(hex string) Color {
	return NewColor(ColorConfig{Hex: hex})
}

func NewANSI(code int) Color {
	return NewColor(ColorConfig{ANSI: code})
}

func NewColor256(code int) Color {
	return NewColor(ColorConfig{Color256: code})
}

func RGB(r, g, b uint8) Color { return NewRGB(r, g, b) }
func Hex(hex string) Color    { return NewHex(hex) }
func ANSIFunc(code int) Color { return NewANSI(code) }
func Color256Func(code int) Color {
	return NewColor256(code)
}

// NewColor creates a Color from a config plus optional functional options.
func NewColor(cfg ColorConfig, opts ...share.Option[ColorConfig]) Color {
	share.ApplyOptions(&cfg, opts...)

	if cfg.Hex != "" {
		parseHexIntoConfig(&cfg)
	}
	if cfg.Hex == "" && cfg.R == 0 && cfg.G == 0 && cfg.B == 0 {
		switch {
		case cfg.ANSI != 0:
			cfg.R, cfg.G, cfg.B = ansiToRGB(cfg.ANSI)
		case cfg.Color256 != 0:
			cfg.R, cfg.G, cfg.B = color256ToRGB(cfg.Color256)
		}
	}

	if cfg.Hex == "" {
		cfg.Hex = fmt.Sprintf("#%02X%02X%02X", cfg.R, cfg.G, cfg.B)
	}
	if cfg.ANSI == 0 && (cfg.R != 0 || cfg.G != 0 || cfg.B != 0) {
		cfg.ANSI = rgbToANSI(cfg.R, cfg.G, cfg.B)
	}
	if cfg.Color256 == 0 && (cfg.R != 0 || cfg.G != 0 || cfg.B != 0) {
		cfg.Color256 = rgbTo256(cfg.R, cfg.G, cfg.B)
	}

	return Color{
		R:        cfg.R,
		G:        cfg.G,
		B:        cfg.B,
		ANSI:     cfg.ANSI,
		Color256: cfg.Color256,
		Hex:      cfg.Hex,
		Name:     cfg.Name,
		IsBg:     cfg.IsBg,
	}
}

func NewColorWith(opts ...share.Option[ColorConfig]) Color {
	cfg := DefaultColorConfig()
	return NewColor(cfg, opts...)
}

func WithRGB(r, g, b uint8) share.Option[ColorConfig] {
	return func(cfg *ColorConfig) {
		cfg.R = r
		cfg.G = g
		cfg.B = b
	}
}

func WithHex(hex string) share.Option[ColorConfig] {
	return func(cfg *ColorConfig) {
		cfg.Hex = hex
	}
}

func WithANSI(code int) share.Option[ColorConfig] {
	return func(cfg *ColorConfig) {
		cfg.ANSI = code
	}
}

func WithColor256(code int) share.Option[ColorConfig] {
	return func(cfg *ColorConfig) {
		cfg.Color256 = code
	}
}

func WithName(name string) share.Option[ColorConfig] {
	return func(cfg *ColorConfig) {
		cfg.Name = name
	}
}

func WithBackground() share.Option[ColorConfig] {
	return func(cfg *ColorConfig) {
		cfg.IsBg = true
	}
}

func WithMode(mode Mode) share.Option[ColorConfig] {
	return func(cfg *ColorConfig) {
		cfg.Mode = mode
	}
}

// Render returns the ANSI escape sequence for the given mode.
func (c Color) Render(mode Mode) string {
	if c.IsBg {
		return c.Background(mode)
	}

	switch mode {
	case ModeNoColor:
		return ""
	case ModeANSI:
		if c.ANSI >= 0 && c.ANSI <= 7 {
			return fmt.Sprintf("\033[3%dm", c.ANSI)
		}
		if c.ANSI >= 8 && c.ANSI <= 15 {
			return fmt.Sprintf("\033[9%dm", c.ANSI-8)
		}
		return ""
	case Mode256Color:
		return fmt.Sprintf("\033[38;5;%dm", c.Color256)
	case ModeTrueColor:
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B)
	default:
		return ""
	}
}

// Background returns the background version of this color.
func (c Color) Background(mode Mode) string {
	switch mode {
	case ModeNoColor:
		return ""
	case ModeANSI:
		if c.ANSI >= 0 && c.ANSI <= 7 {
			return fmt.Sprintf("\033[4%dm", c.ANSI)
		}
		if c.ANSI >= 8 && c.ANSI <= 15 {
			return fmt.Sprintf("\033[10%dm", c.ANSI-8)
		}
		return ""
	case Mode256Color:
		return fmt.Sprintf("\033[48;5;%dm", c.Color256)
	case ModeTrueColor:
		return fmt.Sprintf("\033[48;2;%d;%d;%dm", c.R, c.G, c.B)
	default:
		return ""
	}
}

// Bg returns a background version of this color.
func (c Color) Bg() Color {
	return Color{
		R:        c.R,
		G:        c.G,
		B:        c.B,
		ANSI:     c.ANSI,
		Color256: c.Color256,
		Hex:      c.Hex,
		Name:     c.Name + "_bg",
		IsBg:     true,
	}
}

// WithName returns a copy of the color with a new name.
func (c Color) WithName(name string) Color {
	c.Name = name
	return c
}

func (c Color) String() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Hex
}

func (c Color) Apply(text string) string {
	return c.Render(ModeTrueColor) + text + Reset
}

func (c Color) ApplyMode(text string, mode Mode) string {
	if mode == ModeNoColor {
		return text
	}
	return c.Render(mode) + text + Reset
}

func parseHexIntoConfig(cfg *ColorConfig) {
	hex := strings.TrimPrefix(cfg.Hex, "#")
	if len(hex) == 3 {
		hex = string(
			hex[0],
		) + string(
			hex[0],
		) + string(
			hex[1],
		) + string(
			hex[1],
		) + string(
			hex[2],
		) + string(
			hex[2],
		)
	}
	if len(hex) != 6 {
		return
	}

	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)

	cfg.R = uint8(r)
	cfg.G = uint8(g)
	cfg.B = uint8(b)
	cfg.Hex = "#" + strings.ToUpper(hex)
}

func rgbToANSI(r, g, b uint8) int {
	brightness := (int(r) + int(g) + int(b)) / 3
	maxVal := max(r, g, b)

	if maxVal == 0 {
		return 0
	}

	bright := brightness > 127

	switch {
	case r == maxVal && g > b:
		if bright {
			return 11
		}
		return 3
	case r == maxVal:
		if bright {
			return 9
		}
		return 1
	case g == maxVal && b > r:
		if bright {
			return 14
		}
		return 6
	case g == maxVal:
		if bright {
			return 10
		}
		return 2
	case b == maxVal && r > g:
		if bright {
			return 13
		}
		return 5
	case b == maxVal:
		if bright {
			return 12
		}
		return 4
	default:
		if bright {
			return 15
		}
		return 7
	}
}

func rgbTo256(r, g, b uint8) int {
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return 232 + int(r-8)/10
	}

	r6 := int(r) * 5 / 255
	g6 := int(g) * 5 / 255
	b6 := int(b) * 5 / 255

	return 16 + 36*r6 + 6*g6 + b6
}

func ansiToRGB(ansi int) (uint8, uint8, uint8) {
	colors := [][]uint8{
		{0, 0, 0},
		{128, 0, 0},
		{0, 128, 0},
		{128, 128, 0},
		{0, 0, 128},
		{128, 0, 128},
		{0, 128, 128},
		{192, 192, 192},
		{128, 128, 128},
		{255, 0, 0},
		{0, 255, 0},
		{255, 255, 0},
		{0, 0, 255},
		{255, 0, 255},
		{0, 255, 255},
		{255, 255, 255},
	}

	if ansi >= 0 && ansi < len(colors) {
		return colors[ansi][0], colors[ansi][1], colors[ansi][2]
	}
	return 0, 0, 0
}

func color256ToRGB(code int) (uint8, uint8, uint8) {
	if code < 16 {
		return ansiToRGB(code)
	}
	if code < 232 {
		code -= 16
		r := code / 36
		g := (code % 36) / 6
		b := code % 6

		rVal := clampUint8(r * 255 / 5)
		gVal := clampUint8(g * 255 / 5)
		bVal := clampUint8(b * 255 / 5)

		return rVal, gVal, bVal
	}

	gray := clampUint8(8 + (code-232)*10)
	return gray, gray, gray
}

func clampUint8(value int) uint8 {
	switch {
	case value <= 0:
		return 0
	case value >= 255:
		return 255
	default:
		return uint8(value)
	}
}

func max(a, b, c uint8) uint8 {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}
