package color

import (
	"fmt"
	"strings"

	"github.com/oxhq/tfx/internal/share"
)

// StyleConfig provides structured configuration for text styling.
type StyleConfig struct {
	Text       string
	ForeGround Color
	Background Color
	Bold       bool
	Dim        bool
	Italic     bool
	Underline  bool
	Blink      bool
	Reverse    bool
	Strike     bool
	Mode       Mode
}

// DefaultStyleConfig returns default styling configuration.
func DefaultStyleConfig() StyleConfig {
	return StyleConfig{Mode: ModeTrueColor}
}

func Style(text string, fg Color) string {
	return NewStyle(StyleConfig{Text: text, ForeGround: fg})
}

func StyleBg(text string, fg, bg Color) string {
	return NewStyle(StyleConfig{Text: text, ForeGround: fg, Background: bg})
}

func NewStyle(cfg StyleConfig, opts ...share.Option[StyleConfig]) string {
	share.ApplyOptions(&cfg, opts...)
	return renderStyledText(cfg)
}

func NewStyleWith(opts ...share.Option[StyleConfig]) string {
	cfg := DefaultStyleConfig()
	return NewStyle(cfg, opts...)
}

func WithText(text string) share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Text = text
	}
}

func WithForeground(color Color) share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.ForeGround = color
	}
}

func WithBg(color Color) share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Background = color
	}
}

func WithBold() share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Bold = true
	}
}

func WithDim() share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Dim = true
	}
}

func WithItalic() share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Italic = true
	}
}

func WithUnderline() share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Underline = true
	}
}

func WithBlink() share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Blink = true
	}
}

func WithReverse() share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Reverse = true
	}
}

func WithStrike() share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Strike = true
	}
}

func WithStyleMode(mode Mode) share.Option[StyleConfig] {
	return func(cfg *StyleConfig) {
		cfg.Mode = mode
	}
}

func Apply(text string, color Color, mode Mode) string {
	if mode == ModeNoColor {
		return text
	}
	return color.Render(mode) + text + Reset
}

func ApplyBg(text string, fg, bg Color, mode Mode) string {
	if mode == ModeNoColor {
		return text
	}
	return fg.Render(mode) + bg.Background(mode) + text + Reset
}

func Sprint(color Color, args ...any) string {
	text := fmt.Sprint(args...)
	return Apply(text, color, ModeTrueColor)
}

func Sprintf(color Color, format string, args ...any) string {
	text := fmt.Sprintf(format, args...)
	return Apply(text, color, ModeTrueColor)
}

func Combine(codes ...string) string {
	return strings.Join(codes, "")
}

func GradientText(text string, colors []Color, mode Mode) string {
	if len(colors) == 0 || len(text) == 0 {
		return text
	}
	if len(colors) == 1 {
		return Apply(text, colors[0], mode)
	}

	result := ""
	textLen := len(text)
	colorLen := len(colors)

	for i, char := range text {
		colorIndex := (i * colorLen) / textLen
		if colorIndex >= colorLen {
			colorIndex = colorLen - 1
		}
		result += Apply(string(char), colors[colorIndex], mode)
	}

	return result
}

func RainbowText(text string, mode Mode) string {
	rainbowColors := []Color{ColorRed, ColorYellow, ColorGreen, ColorCyan, ColorBlue, ColorMagenta}
	return GradientText(text, rainbowColors, mode)
}

func PulseText(text string, color Color, pulse bool) string {
	if pulse {
		return Combine(Dim, color.Render(ModeTrueColor)) + text + Reset
	}
	return Apply(text, color, ModeTrueColor)
}

func StripANSI(text string) string {
	result := ""
	inEscape := false
	runes := []rune(text)

	for i := range runes {
		if runes[i] == '\033' && i+1 < len(runes) && runes[i+1] == '[' {
			inEscape = true
			continue
		}
		if inEscape && runes[i] == 'm' {
			inEscape = false
			continue
		}
		if !inEscape {
			result += string(runes[i])
		}
	}

	return result
}

func GetLength(text string) int {
	return len([]rune(StripANSI(text)))
}

func PadString(text string, width int, padChar rune) string {
	textLen := GetLength(text)
	if textLen >= width {
		return text
	}

	padding := strings.Repeat(string(padChar), width-textLen)
	return text + padding
}

func CenterString(text string, width int) string {
	textLen := GetLength(text)
	if textLen >= width {
		return text
	}

	leftPad := (width - textLen) / 2
	rightPad := width - textLen - leftPad

	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}

func renderStyledText(cfg StyleConfig) string {
	if cfg.Text == "" {
		return ""
	}

	if cfg.Mode == ModeNoColor {
		return cfg.Text
	}

	var codes []string

	if cfg.Bold {
		codes = append(codes, Bold)
	}
	if cfg.Dim {
		codes = append(codes, Dim)
	}
	if cfg.Italic {
		codes = append(codes, Italic)
	}
	if cfg.Underline {
		codes = append(codes, Underline)
	}
	if cfg.Blink {
		codes = append(codes, Blink)
	}
	if cfg.Reverse {
		codes = append(codes, Reverse)
	}
	if cfg.Strike {
		codes = append(codes, Strike)
	}

	if cfg.ForeGround != (Color{}) {
		codes = append(codes, cfg.ForeGround.Render(cfg.Mode))
	}
	if cfg.Background != (Color{}) {
		codes = append(codes, cfg.Background.Background(cfg.Mode))
	}

	if len(codes) == 0 {
		return cfg.Text
	}

	return Combine(codes...) + cfg.Text + Reset
}

func Success(text string) string {
	return Apply(text, ColorSuccess, ModeTrueColor)
}

func Error(text string) string {
	return Apply(text, ColorError, ModeTrueColor)
}

func Warning(text string) string {
	return Apply(text, ColorWarning, ModeTrueColor)
}

func Info(text string) string {
	return Apply(text, ColorInfo, ModeTrueColor)
}

func Debug(text string) string {
	return Apply(text, ColorDebug, ModeTrueColor)
}

func Badge(text string, fg, bg Color) string {
	return ApplyBg(" "+text+" ", fg, bg, ModeTrueColor)
}

func SuccessBadge(text string) string {
	return Badge(text, ColorBlack, ColorSuccess)
}

func ErrorBadge(text string) string {
	return Badge(text, ColorWhite, ColorError)
}

func WarningBadge(text string) string {
	return Badge(text, ColorBlack, ColorWarning)
}

func InfoBadge(text string) string {
	return Badge(text, ColorWhite, ColorInfo)
}

func DebugBadge(text string) string {
	return Badge(text, ColorWhite, ColorDebug)
}

func ProgressBar(current, total, width int, filledColor, emptyColor Color) string {
	if total == 0 {
		total = 1
	}

	filled := min((current*width)/total, width)
	filledStr := strings.Repeat("█", filled)
	emptyStr := strings.Repeat("░", width-filled)

	return Apply(filledStr, filledColor, ModeTrueColor) + Apply(emptyStr, emptyColor, ModeTrueColor)
}

func Border(text string, borderColor Color) string {
	lines := strings.Split(text, "\n")
	maxWidth := 0

	for _, line := range lines {
		if w := GetLength(line); w > maxWidth {
			maxWidth = w
		}
	}

	result := Apply("┌"+strings.Repeat("─", maxWidth+2)+"┐", borderColor, ModeTrueColor) + "\n"

	for _, line := range lines {
		padding := strings.Repeat(" ", maxWidth-GetLength(line))
		result += Apply(
			"│",
			borderColor,
			ModeTrueColor,
		) + " " + line + padding + " " + Apply(
			"│",
			borderColor,
			ModeTrueColor,
		) + "\n"
	}

	result += Apply("└"+strings.Repeat("─", maxWidth+2)+"┘", borderColor, ModeTrueColor)
	return result
}
