package color

import (
	"maps"

	"github.com/oxhq/tfx/internal/share"
)

// Palette represents a collection of named colors.
type Palette map[string]Color

// PaletteConfig provides structured configuration for Palette creation.
type PaletteConfig struct {
	Name   string
	Colors map[string]Color
}

// DefaultPaletteConfig returns default configuration for Palette.
func DefaultPaletteConfig() PaletteConfig {
	return PaletteConfig{
		Name:   "default",
		Colors: make(map[string]Color),
	}
}

func BasicPalette() Palette {
	return StatusPalette()
}

func NewPalette(cfg PaletteConfig, opts ...share.Option[PaletteConfig]) Palette {
	share.ApplyOptions(&cfg, opts...)
	return Palette(cfg.Colors)
}

func NewPaletteWith(opts ...share.Option[PaletteConfig]) Palette {
	cfg := DefaultPaletteConfig()
	return NewPalette(cfg, opts...)
}

func WithPaletteName(name string) share.Option[PaletteConfig] {
	return func(cfg *PaletteConfig) {
		cfg.Name = name
	}
}

func WithColors(colors map[string]Color) share.Option[PaletteConfig] {
	return func(cfg *PaletteConfig) {
		if cfg.Colors == nil {
			cfg.Colors = make(map[string]Color)
		}
		maps.Copy(cfg.Colors, colors)
	}
}

func WithColor(name string, color Color) share.Option[PaletteConfig] {
	return func(cfg *PaletteConfig) {
		if cfg.Colors == nil {
			cfg.Colors = make(map[string]Color)
		}
		cfg.Colors[name] = color
	}
}

func (p Palette) Get(name string) (Color, bool) {
	color, exists := p[name]
	return color, exists
}

func (p Palette) Set(name string, color Color) {
	p[name] = color
}

func (p Palette) Names() []string {
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}
	return names
}

func (p Palette) Merge(other Palette) Palette {
	merged := make(Palette, len(p)+len(other))
	maps.Copy(merged, p)
	maps.Copy(merged, other)
	return merged
}

var (
	ColorBlack   = NewANSI(0).WithName("black")
	ColorRed     = NewANSI(1).WithName("red")
	ColorGreen   = NewANSI(2).WithName("green")
	ColorYellow  = NewANSI(3).WithName("yellow")
	ColorBlue    = NewANSI(4).WithName("blue")
	ColorMagenta = NewANSI(5).WithName("magenta")
	ColorCyan    = NewANSI(6).WithName("cyan")
	ColorWhite   = NewANSI(7).WithName("white")

	ColorBrightBlack   = NewANSI(8).WithName("bright_black")
	ColorBrightRed     = NewANSI(9).WithName("bright_red")
	ColorBrightGreen   = NewANSI(10).WithName("bright_green")
	ColorBrightYellow  = NewANSI(11).WithName("bright_yellow")
	ColorBrightBlue    = NewANSI(12).WithName("bright_blue")
	ColorBrightMagenta = NewANSI(13).WithName("bright_magenta")
	ColorBrightCyan    = NewANSI(14).WithName("bright_cyan")
	ColorBrightWhite   = NewANSI(15).WithName("bright_white")

	ColorSuccess = ColorBrightGreen.WithName("success")
	ColorError   = ColorBrightRed.WithName("error")
	ColorWarning = ColorBrightYellow.WithName("warning")
	ColorInfo    = ColorBrightCyan.WithName("info")
	ColorDebug   = ColorBrightMagenta.WithName("debug")
)

var (
	MaterialRed        = NewHex("#F44336").WithName("material_red")
	MaterialPink       = NewHex("#E91E63").WithName("material_pink")
	MaterialPurple     = NewHex("#9C27B0").WithName("material_purple")
	MaterialDeepPurple = NewHex("#673AB7").WithName("material_deep_purple")
	MaterialIndigo     = NewHex("#3F51B5").WithName("material_indigo")
	MaterialBlue       = NewHex("#2196F3").WithName("material_blue")
	MaterialLightBlue  = NewHex("#03A9F4").WithName("material_light_blue")
	MaterialCyan       = NewHex("#00BCD4").WithName("material_cyan")
	MaterialTeal       = NewHex("#009688").WithName("material_teal")
	MaterialGreen      = NewHex("#4CAF50").WithName("material_green")
	MaterialLightGreen = NewHex("#8BC34A").WithName("material_light_green")
	MaterialLime       = NewHex("#CDDC39").WithName("material_lime")
	MaterialYellow     = NewHex("#FFEB3B").WithName("material_yellow")
	MaterialAmber      = NewHex("#FFC107").WithName("material_amber")
	MaterialOrange     = NewHex("#FF9800").WithName("material_orange")
	MaterialDeepOrange = NewHex("#FF5722").WithName("material_deep_orange")
)

var (
	TailwindRed     = NewHex("#EF4444").WithName("tailwind_red")
	TailwindOrange  = NewHex("#F97316").WithName("tailwind_orange")
	TailwindAmber   = NewHex("#F59E0B").WithName("tailwind_amber")
	TailwindYellow  = NewHex("#EAB308").WithName("tailwind_yellow")
	TailwindLime    = NewHex("#84CC16").WithName("tailwind_lime")
	TailwindGreen   = NewHex("#22C55E").WithName("tailwind_green")
	TailwindEmerald = NewHex("#10B981").WithName("tailwind_emerald")
	TailwindTeal    = NewHex("#14B8A6").WithName("tailwind_teal")
	TailwindCyan    = NewHex("#06B6D4").WithName("tailwind_cyan")
	TailwindSky     = NewHex("#0EA5E9").WithName("tailwind_sky")
	TailwindBlue    = NewHex("#3B82F6").WithName("tailwind_blue")
	TailwindIndigo  = NewHex("#6366F1").WithName("tailwind_indigo")
	TailwindViolet  = NewHex("#8B5CF6").WithName("tailwind_violet")
	TailwindPurple  = NewHex("#A855F7").WithName("tailwind_purple")
	TailwindFuchsia = NewHex("#D946EF").WithName("tailwind_fuchsia")
	TailwindPink    = NewHex("#EC4899").WithName("tailwind_pink")
	TailwindRose    = NewHex("#F43F5E").WithName("tailwind_rose")
)

var (
	DraculaPurple = NewHex("#BD93F9").WithName("dracula_purple")
	DraculaPink   = NewHex("#FF79C6").WithName("dracula_pink")
	DraculaGreen  = NewHex("#50FA7B").WithName("dracula_green")
	DraculaOrange = NewHex("#FFB86C").WithName("dracula_orange")
	DraculaRed    = NewHex("#FF5555").WithName("dracula_red")
	DraculaYellow = NewHex("#F1FA8C").WithName("dracula_yellow")
	DraculaCyan   = NewHex("#8BE9FD").WithName("dracula_cyan")
)

var (
	NordBlue      = NewHex("#5E81AC").WithName("nord_blue")
	NordLightBlue = NewHex("#81A1C1").WithName("nord_light_blue")
	NordCyan      = NewHex("#88C0D0").WithName("nord_cyan")
	NordGreen     = NewHex("#A3BE8C").WithName("nord_green")
	NordYellow    = NewHex("#EBCB8B").WithName("nord_yellow")
	NordOrange    = NewHex("#D08770").WithName("nord_orange")
	NordRed       = NewHex("#BF616A").WithName("nord_red")
	NordPurple    = NewHex("#B48EAD").WithName("nord_purple")
)

var (
	GithubGreenLight  = NewHex("#28A745").WithName("github_green_light")
	GithubGreenDark   = NewHex("#22863A").WithName("github_green_dark")
	GithubRedLight    = NewHex("#D73A49").WithName("github_red_light")
	GithubRedDark     = NewHex("#B31D28").WithName("github_red_dark")
	GithubBlueLight   = NewHex("#0366D6").WithName("github_blue_light")
	GithubBlueDark    = NewHex("#005CC5").WithName("github_blue_dark")
	GithubOrangeLight = NewHex("#F66A0A").WithName("github_orange_light")
	GithubOrangeDark  = NewHex("#E36209").WithName("github_orange_dark")
	GithubPurple      = NewHex("#6F42C1").WithName("github_purple")
	GithubYellow      = NewHex("#FFD33D").WithName("github_yellow")
)

var (
	VSCodeBlue   = NewHex("#007ACC").WithName("vscode_blue")
	VSCodeGreen  = NewHex("#608B4E").WithName("vscode_green")
	VSCodeRed    = NewHex("#F44747").WithName("vscode_red")
	VSCodeOrange = NewHex("#FF8C00").WithName("vscode_orange")
	VSCodePurple = NewHex("#C586C0").WithName("vscode_purple")
	VSCodeYellow = NewHex("#FFCD3C").WithName("vscode_yellow")
	VSCodeCyan   = NewHex("#4EC9B0").WithName("vscode_cyan")
)

// ColorTheme represents a theme with semantic colors for logging.
type ColorTheme struct {
	Name      string
	Success   Color
	Error     Color
	Warning   Color
	Info      Color
	Debug     Color
	Primary   Color
	Secondary Color
	Accent    Color
}

var (
	ModernGreen  = NewHex("#10B981").WithName("modern_green")
	ModernBlue   = NewHex("#3B82F6").WithName("modern_blue")
	ModernRed    = NewHex("#EF4444").WithName("modern_red")
	ModernYellow = NewHex("#F59E0B").WithName("modern_yellow")
	ModernPurple = NewHex("#8B5CF6").WithName("modern_purple")
	ModernCyan   = NewHex("#06B6D4").WithName("modern_cyan")
	ModernOrange = NewHex("#F97316").WithName("modern_orange")
	ModernPink   = NewHex("#EC4899").WithName("modern_pink")
	ModernGray   = NewHex("#6B7280").WithName("modern_gray")
	ModernSlate  = NewHex("#475569").WithName("modern_slate")
)

var DefaultTheme = ColorTheme{
	Name:      "default",
	Success:   ModernGreen,
	Error:     ModernRed,
	Warning:   ModernYellow,
	Info:      ModernBlue,
	Debug:     ModernPurple,
	Primary:   ModernBlue,
	Secondary: ModernCyan,
	Accent:    ModernPink,
}

var DraculaTheme = ColorTheme{
	Name:      "dracula",
	Success:   DraculaGreen,
	Error:     DraculaRed,
	Warning:   DraculaYellow,
	Info:      DraculaCyan,
	Debug:     DraculaPurple,
	Primary:   DraculaPurple,
	Secondary: DraculaPink,
	Accent:    DraculaOrange,
}

var MaterialTheme = ColorTheme{
	Name:      "material",
	Success:   MaterialGreen,
	Error:     MaterialRed,
	Warning:   MaterialAmber,
	Info:      MaterialBlue,
	Debug:     MaterialPurple,
	Primary:   MaterialBlue,
	Secondary: MaterialCyan,
	Accent:    MaterialPink,
}

var NordTheme = ColorTheme{
	Name:      "nord",
	Success:   NordGreen,
	Error:     NordRed,
	Warning:   NordYellow,
	Info:      NordCyan,
	Debug:     NordPurple,
	Primary:   NordBlue,
	Secondary: NordLightBlue,
	Accent:    NordOrange,
}

var GitHubTheme = ColorTheme{
	Name:      "github",
	Success:   GithubGreenLight,
	Error:     GithubRedLight,
	Warning:   GithubYellow,
	Info:      GithubBlueLight,
	Debug:     GithubPurple,
	Primary:   GithubBlueLight,
	Secondary: GithubBlueDark,
	Accent:    GithubOrangeLight,
}

func StatusPalette() Palette {
	return Palette{
		"success":  ColorSuccess,
		"error":    ColorError,
		"warning":  ColorWarning,
		"info":     ColorInfo,
		"debug":    ColorDebug,
		"critical": MaterialDeepOrange.WithName("critical"),
		"notice":   MaterialCyan.WithName("notice"),
	}
}

func MaterialPalette() Palette {
	return Palette{
		"red":         MaterialRed,
		"pink":        MaterialPink,
		"purple":      MaterialPurple,
		"deep_purple": MaterialDeepPurple,
		"indigo":      MaterialIndigo,
		"blue":        MaterialBlue,
		"light_blue":  MaterialLightBlue,
		"cyan":        MaterialCyan,
		"teal":        MaterialTeal,
		"green":       MaterialGreen,
		"light_green": MaterialLightGreen,
		"lime":        MaterialLime,
		"yellow":      MaterialYellow,
		"amber":       MaterialAmber,
		"orange":      MaterialOrange,
		"deep_orange": MaterialDeepOrange,
	}
}

func DraculaPalette() Palette {
	return Palette{
		"purple": DraculaPurple,
		"pink":   DraculaPink,
		"green":  DraculaGreen,
		"orange": DraculaOrange,
		"red":    DraculaRed,
		"yellow": DraculaYellow,
		"cyan":   DraculaCyan,
	}
}

func NordPalette() Palette {
	return Palette{
		"blue":       NordBlue,
		"light_blue": NordLightBlue,
		"cyan":       NordCyan,
		"green":      NordGreen,
		"yellow":     NordYellow,
		"orange":     NordOrange,
		"red":        NordRed,
		"purple":     NordPurple,
	}
}

func TailwindPalette() Palette {
	return Palette{
		"red":     TailwindRed,
		"orange":  TailwindOrange,
		"amber":   TailwindAmber,
		"yellow":  TailwindYellow,
		"lime":    TailwindLime,
		"green":   TailwindGreen,
		"emerald": TailwindEmerald,
		"teal":    TailwindTeal,
		"cyan":    TailwindCyan,
		"sky":     TailwindSky,
		"blue":    TailwindBlue,
		"indigo":  TailwindIndigo,
		"violet":  TailwindViolet,
		"purple":  TailwindPurple,
		"fuchsia": TailwindFuchsia,
		"pink":    TailwindPink,
		"rose":    TailwindRose,
	}
}

func GitHubPalette() Palette {
	return Palette{
		"green_light":  GithubGreenLight,
		"green_dark":   GithubGreenDark,
		"red_light":    GithubRedLight,
		"red_dark":     GithubRedDark,
		"blue_light":   GithubBlueLight,
		"blue_dark":    GithubBlueDark,
		"orange_light": GithubOrangeLight,
		"orange_dark":  GithubOrangeDark,
		"purple":       GithubPurple,
		"yellow":       GithubYellow,
	}
}

func VSCodePalette() Palette {
	return Palette{
		"blue":   VSCodeBlue,
		"green":  VSCodeGreen,
		"red":    VSCodeRed,
		"orange": VSCodeOrange,
		"purple": VSCodePurple,
		"yellow": VSCodeYellow,
		"cyan":   VSCodeCyan,
	}
}

var AllPalettes = map[string]func() Palette{
	"status":   StatusPalette,
	"material": MaterialPalette,
	"dracula":  DraculaPalette,
	"nord":     NordPalette,
	"tailwind": TailwindPalette,
	"github":   GitHubPalette,
	"vscode":   VSCodePalette,
}

func GetPalette(name string) (Palette, bool) {
	paletteFunc, exists := AllPalettes[name]
	if !exists {
		return nil, false
	}
	return paletteFunc(), true
}

func ListPalettes() []string {
	names := make([]string, 0, len(AllPalettes))
	for name := range AllPalettes {
		names = append(names, name)
	}
	return names
}
