package color

// EncodingSystem provides colors in a specific encoding mode.
type EncodingSystem struct {
	Black   Color
	Red     Color
	Green   Color
	Yellow  Color
	Blue    Color
	Magenta Color
	Cyan    Color
	White   Color

	BrightBlack   Color
	BrightRed     Color
	BrightGreen   Color
	BrightYellow  Color
	BrightBlue    Color
	BrightMagenta Color
	BrightCyan    Color
	BrightWhite   Color
}

// ThemeSystem provides colors in a specific theme.
//
// This is a compatibility layer for the legacy global theme surface. New code
// should prefer the semantic ColorTheme and palette helpers in palettes.go.
type ThemeSystem struct {
	Black   Color
	Red     Color
	Green   Color
	Yellow  Color
	Blue    Color
	Magenta Color
	Cyan    Color
	White   Color

	Purple Color
	Orange Color
	Pink   Color
	Teal   Color
	Lime   Color
	Indigo Color
}

var (
	// Legacy compatibility globals backed by the current theme selection.
	ANSI      *EncodingSystem
	TrueColor *EncodingSystem
	Color256  *EncodingSystem

	// Legacy theme handles retained for compatibility with older callers.
	Material *ThemeSystem
	Dracula  *ThemeSystem
	Nord     *ThemeSystem
	GitHub   *ThemeSystem
)

var (
	Black   Color
	Red     Color
	Green   Color
	Yellow  Color
	Blue    Color
	Magenta Color
	Cyan    Color
	White   Color

	Purple Color
	Orange Color
	Pink   Color
	Teal   Color
	Lime   Color
	Indigo Color
)

var (
	currentEncoding = ModeANSI
	currentTheme    = "material"
)

func init() {
	initializeCleanColorSystems()
}

func initializeCleanColorSystems() {
	createCleanEncodingSystems()
	createCleanThemeSystems()
	updateCleanDefaultColors()
}

func createCleanEncodingSystems() {
	ANSI = &EncodingSystem{
		Black:   NewColor(ColorConfig{ANSI: 0, Name: "ansi_black"}),
		Red:     NewColor(ColorConfig{ANSI: 1, Name: "ansi_red"}),
		Green:   NewColor(ColorConfig{ANSI: 2, Name: "ansi_green"}),
		Yellow:  NewColor(ColorConfig{ANSI: 3, Name: "ansi_yellow"}),
		Blue:    NewColor(ColorConfig{ANSI: 4, Name: "ansi_blue"}),
		Magenta: NewColor(ColorConfig{ANSI: 5, Name: "ansi_magenta"}),
		Cyan:    NewColor(ColorConfig{ANSI: 6, Name: "ansi_cyan"}),
		White:   NewColor(ColorConfig{ANSI: 7, Name: "ansi_white"}),

		BrightBlack:   NewColor(ColorConfig{ANSI: 8, Name: "ansi_bright_black"}),
		BrightRed:     NewColor(ColorConfig{ANSI: 9, Name: "ansi_bright_red"}),
		BrightGreen:   NewColor(ColorConfig{ANSI: 10, Name: "ansi_bright_green"}),
		BrightYellow:  NewColor(ColorConfig{ANSI: 11, Name: "ansi_bright_yellow"}),
		BrightBlue:    NewColor(ColorConfig{ANSI: 12, Name: "ansi_bright_blue"}),
		BrightMagenta: NewColor(ColorConfig{ANSI: 13, Name: "ansi_bright_magenta"}),
		BrightCyan:    NewColor(ColorConfig{ANSI: 14, Name: "ansi_bright_cyan"}),
		BrightWhite:   NewColor(ColorConfig{ANSI: 15, Name: "ansi_bright_white"}),
	}

	TrueColor = &EncodingSystem{
		Black:   NewColor(ColorConfig{R: 0, G: 0, B: 0, Name: "true_black"}),
		Red:     NewColor(ColorConfig{R: 255, G: 0, B: 0, Name: "true_red"}),
		Green:   NewColor(ColorConfig{R: 0, G: 255, B: 0, Name: "true_green"}),
		Yellow:  NewColor(ColorConfig{R: 255, G: 255, B: 0, Name: "true_yellow"}),
		Blue:    NewColor(ColorConfig{R: 0, G: 0, B: 255, Name: "true_blue"}),
		Magenta: NewColor(ColorConfig{R: 255, G: 0, B: 255, Name: "true_magenta"}),
		Cyan:    NewColor(ColorConfig{R: 0, G: 255, B: 255, Name: "true_cyan"}),
		White:   NewColor(ColorConfig{R: 255, G: 255, B: 255, Name: "true_white"}),

		BrightBlack:   NewColor(ColorConfig{R: 128, G: 128, B: 128, Name: "true_bright_black"}),
		BrightRed:     NewColor(ColorConfig{R: 255, G: 128, B: 128, Name: "true_bright_red"}),
		BrightGreen:   NewColor(ColorConfig{R: 128, G: 255, B: 128, Name: "true_bright_green"}),
		BrightYellow:  NewColor(ColorConfig{R: 255, G: 255, B: 128, Name: "true_bright_yellow"}),
		BrightBlue:    NewColor(ColorConfig{R: 128, G: 128, B: 255, Name: "true_bright_blue"}),
		BrightMagenta: NewColor(ColorConfig{R: 255, G: 128, B: 255, Name: "true_bright_magenta"}),
		BrightCyan:    NewColor(ColorConfig{R: 128, G: 255, B: 255, Name: "true_bright_cyan"}),
		BrightWhite:   NewColor(ColorConfig{R: 255, G: 255, B: 255, Name: "true_bright_white"}),
	}

	Color256 = &EncodingSystem{
		Black:   NewColor(ColorConfig{Color256: 0, Name: "256_black"}),
		Red:     NewColor(ColorConfig{Color256: 1, Name: "256_red"}),
		Green:   NewColor(ColorConfig{Color256: 2, Name: "256_green"}),
		Yellow:  NewColor(ColorConfig{Color256: 3, Name: "256_yellow"}),
		Blue:    NewColor(ColorConfig{Color256: 4, Name: "256_blue"}),
		Magenta: NewColor(ColorConfig{Color256: 5, Name: "256_magenta"}),
		Cyan:    NewColor(ColorConfig{Color256: 6, Name: "256_cyan"}),
		White:   NewColor(ColorConfig{Color256: 7, Name: "256_white"}),

		BrightBlack:   NewColor(ColorConfig{Color256: 8, Name: "256_bright_black"}),
		BrightRed:     NewColor(ColorConfig{Color256: 9, Name: "256_bright_red"}),
		BrightGreen:   NewColor(ColorConfig{Color256: 10, Name: "256_bright_green"}),
		BrightYellow:  NewColor(ColorConfig{Color256: 11, Name: "256_bright_yellow"}),
		BrightBlue:    NewColor(ColorConfig{Color256: 12, Name: "256_bright_blue"}),
		BrightMagenta: NewColor(ColorConfig{Color256: 13, Name: "256_bright_magenta"}),
		BrightCyan:    NewColor(ColorConfig{Color256: 14, Name: "256_bright_cyan"}),
		BrightWhite:   NewColor(ColorConfig{Color256: 15, Name: "256_bright_white"}),
	}
}

func createCleanThemeSystems() {
	Material = &ThemeSystem{
		Black:   NewHex("#424242").WithName("material_black"),
		Red:     MaterialRed,
		Green:   MaterialGreen,
		Yellow:  MaterialYellow,
		Blue:    MaterialBlue,
		Magenta: MaterialPurple,
		Cyan:    MaterialCyan,
		White:   NewHex("#FFFFFF").WithName("material_white"),
		Purple:  MaterialPurple,
		Orange:  MaterialOrange,
		Pink:    MaterialPink,
		Teal:    MaterialTeal,
		Lime:    MaterialLime,
		Indigo:  MaterialIndigo,
	}

	Dracula = &ThemeSystem{
		Black:   NewHex("#282A36").WithName("dracula_black"),
		Red:     DraculaRed,
		Green:   DraculaGreen,
		Yellow:  DraculaYellow,
		Blue:    NewHex("#6272A4").WithName("dracula_blue"),
		Magenta: DraculaPurple,
		Cyan:    DraculaCyan,
		White:   NewHex("#F8F8F2").WithName("dracula_white"),
		Purple:  DraculaPurple,
		Orange:  DraculaOrange,
		Pink:    DraculaPink,
		Teal:    DraculaCyan,
		Lime:    DraculaGreen,
		Indigo:  DraculaPurple,
	}

	Nord = &ThemeSystem{
		Black:   NewHex("#2E3440").WithName("nord_black"),
		Red:     NordRed,
		Green:   NordGreen,
		Yellow:  NordYellow,
		Blue:    NordBlue,
		Magenta: NordPurple,
		Cyan:    NordCyan,
		White:   NewHex("#ECEFF4").WithName("nord_white"),
		Purple:  NordPurple,
		Orange:  NordOrange,
		Pink:    NordPurple,
		Teal:    NordCyan,
		Lime:    NordGreen,
		Indigo:  NordBlue,
	}

	GitHub = &ThemeSystem{
		Black:   NewHex("#24292e").WithName("github_black"),
		Red:     GithubRedLight,
		Green:   GithubGreenLight,
		Yellow:  NewHex("#FBBF40").WithName("github_yellow"),
		Blue:    GithubBlueLight,
		Magenta: NewHex("#B392F0").WithName("github_magenta"),
		Cyan:    GithubBlueLight,
		White:   NewHex("#FFFFFF").WithName("github_white"),
		Purple:  NewHex("#B392F0").WithName("github_purple"),
		Orange:  GithubOrangeLight,
		Pink:    NewHex("#F97583").WithName("github_pink"),
		Teal:    GithubBlueLight,
		Lime:    GithubGreenLight,
		Indigo:  GithubBlueLight,
	}
}

func updateCleanDefaultColors() {
	var source *ThemeSystem

	switch currentTheme {
	case "material":
		source = Material
	case "dracula":
		source = Dracula
	case "nord":
		source = Nord
	case "github":
		source = GitHub
	default:
		source = Material
	}

	Black = source.Black
	Red = source.Red
	Green = source.Green
	Yellow = source.Yellow
	Blue = source.Blue
	Magenta = source.Magenta
	Cyan = source.Cyan
	White = source.White
	Purple = source.Purple
	Orange = source.Orange
	Pink = source.Pink
	Teal = source.Teal
	Lime = source.Lime
	Indigo = source.Indigo
}

func SetDefaultEncoding(mode Mode) {
	currentEncoding = mode
}

func SetDefaultTheme(theme string) {
	currentTheme = theme
	updateCleanDefaultColors()
}

func GetDefaultEncoding() Mode {
	return currentEncoding
}

func GetDefaultTheme() string {
	return currentTheme
}

func UseMaterial() { SetDefaultTheme("material") }
func UseDracula()  { SetDefaultTheme("dracula") }
func UseNord()     { SetDefaultTheme("nord") }
func UseGitHub()   { SetDefaultTheme("github") }
