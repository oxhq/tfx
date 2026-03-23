package color

import (
	"strings"
	"testing"
)

func TestPaletteHelpersAndLookup(t *testing.T) {
	t.Parallel()

	palette := NewPaletteWith(
		WithPaletteName("custom"),
		WithColor("one", NewRGB(1, 2, 3)),
		WithColors(map[string]Color{"two": NewHex("#040506")}),
	)
	if _, ok := palette.Get("one"); !ok {
		t.Fatal("expected palette Get() to find inserted color")
	}
	palette.Set("three", NewANSI(4))
	if names := palette.Names(); len(names) != 3 {
		t.Fatalf("expected 3 palette names, got %d", len(names))
	}
	merged := palette.Merge(Palette{"four": NewANSI(5)})
	if _, ok := merged.Get("four"); !ok {
		t.Fatal("expected Merge() to include other palette entries")
	}
	if _, ok := GetPalette("material"); !ok {
		t.Fatal("expected named palette lookup to succeed")
	}
	if _, ok := GetPalette("missing"); ok {
		t.Fatal("expected missing palette lookup to fail")
	}
	if len(ListPalettes()) == 0 || len(BasicPalette()) == 0 {
		t.Fatal("expected palette registries to be populated")
	}
}

func TestStyleAndUtilityHelpers(t *testing.T) {
	t.Parallel()

	cfg := DefaultStyleConfig()
	if cfg.Mode != ModeTrueColor {
		t.Fatalf("unexpected default style config: %+v", cfg)
	}

	styled := NewStyleWith(
		WithText("hello"),
		WithForeground(ColorRed),
		WithBg(ColorBlue),
		WithBold(),
		WithDim(),
		WithItalic(),
		WithUnderline(),
		WithBlink(),
		WithReverse(),
		WithStrike(),
		WithStyleMode(ModeANSI),
	)
	if !strings.Contains(styled, "hello") || !strings.Contains(styled, Reset) {
		t.Fatalf("expected styled text with reset, got %q", styled)
	}

	if got := Style("hi", ColorGreen); !strings.Contains(got, "hi") {
		t.Fatalf("unexpected Style() result: %q", got)
	}
	if got := StyleBg("hi", ColorGreen, ColorBlue); !strings.Contains(got, "hi") {
		t.Fatalf("unexpected StyleBg() result: %q", got)
	}
	if got := Apply("x", ColorRed, ModeNoColor); got != "x" {
		t.Fatalf("expected no-color apply to return raw text, got %q", got)
	}
	if got := ApplyBg("x", ColorRed, ColorBlue, ModeNoColor); got != "x" {
		t.Fatalf("expected no-color background apply to return raw text, got %q", got)
	}
	if got := Sprint(ColorGreen, "a", 1); !strings.Contains(got, "a1") {
		t.Fatalf("unexpected Sprint() result: %q", got)
	}
	if got := Sprintf(ColorGreen, "%s-%d", "a", 1); !strings.Contains(got, "a-1") {
		t.Fatalf("unexpected Sprintf() result: %q", got)
	}
	if got := Combine("a", "b", "c"); got != "abc" {
		t.Fatalf("unexpected Combine() result: %q", got)
	}
	if got := GradientText("abc", []Color{ColorRed, ColorBlue}, ModeNoColor); got != "abc" {
		t.Fatalf("expected no-color gradient to preserve text, got %q", got)
	}
	if got := RainbowText("abc", ModeANSI); !strings.Contains(got, "a") {
		t.Fatalf("unexpected RainbowText() result: %q", got)
	}
	if got := PulseText("abc", ColorRed, true); !strings.Contains(got, "abc") {
		t.Fatalf("unexpected PulseText() result: %q", got)
	}
	if got := StripANSI(ColorRed.Render(ModeANSI) + "abc" + Reset); got != "abc" {
		t.Fatalf("unexpected StripANSI() result: %q", got)
	}
	if got := GetLength(ColorRed.Render(ModeANSI) + "abc" + Reset); got != 3 {
		t.Fatalf("unexpected GetLength() result: %d", got)
	}
	if got := PadString("abc", 5, '.'); got != "abc.." {
		t.Fatalf("unexpected PadString() result: %q", got)
	}
	if got := CenterString("abc", 5); got != " abc " {
		t.Fatalf("unexpected CenterString() result: %q", got)
	}
	if got := Success("ok"); !strings.Contains(got, "ok") {
		t.Fatalf("unexpected Success() result: %q", got)
	}
	if got := Error("bad"); !strings.Contains(got, "bad") {
		t.Fatalf("unexpected Error() result: %q", got)
	}
	if got := Warning("warn"); !strings.Contains(got, "warn") {
		t.Fatalf("unexpected Warning() result: %q", got)
	}
	if got := Info("info"); !strings.Contains(got, "info") {
		t.Fatalf("unexpected Info() result: %q", got)
	}
	if got := Debug("debug"); !strings.Contains(got, "debug") {
		t.Fatalf("unexpected Debug() result: %q", got)
	}
	if got := Badge("x", ColorWhite, ColorBlue); !strings.Contains(got, " x ") {
		t.Fatalf("unexpected Badge() result: %q", got)
	}
	if got := SuccessBadge("ok"); !strings.Contains(got, " ok ") {
		t.Fatalf("unexpected SuccessBadge() result: %q", got)
	}
	if got := ErrorBadge("bad"); !strings.Contains(got, " bad ") {
		t.Fatalf("unexpected ErrorBadge() result: %q", got)
	}
	if got := WarningBadge("warn"); !strings.Contains(got, " warn ") {
		t.Fatalf("unexpected WarningBadge() result: %q", got)
	}
	if got := InfoBadge("info"); !strings.Contains(got, " info ") {
		t.Fatalf("unexpected InfoBadge() result: %q", got)
	}
	if got := DebugBadge("dbg"); !strings.Contains(got, " dbg ") {
		t.Fatalf("unexpected DebugBadge() result: %q", got)
	}
	if got := ProgressBar(1, 2, 4, ColorGreen, ColorRed); StripANSI(got) != "██░░" {
		t.Fatalf("unexpected ProgressBar() result: %q", StripANSI(got))
	}
	if got := Border("hi\nok", ColorBlue); !strings.Contains(StripANSI(got), "┌") ||
		!strings.Contains(StripANSI(got), "ok") {
		t.Fatalf("unexpected Border() result: %q", got)
	}
}

func TestColorBuilderHelpers(t *testing.T) {
	t.Parallel()

	cfg := DefaultColorConfig()
	if cfg.Mode != ModeTrueColor {
		t.Fatalf("unexpected default color config: %+v", cfg)
	}

	c := NewColorWith(
		WithRGB(1, 2, 3),
		WithHex("#010203"),
		WithANSI(4),
		WithColor256(5),
		WithName("named"),
		WithBackground(),
		WithMode(ModeANSI),
	)
	if !c.IsBg || c.Name != "named" {
		t.Fatalf("unexpected built color: %+v", c)
	}
	if got := c.Bg(); !got.IsBg || !strings.Contains(got.Name, "_bg") {
		t.Fatalf("unexpected Bg() color: %+v", got)
	}
	if got := c.String(); got != "named" {
		t.Fatalf("unexpected Color.String(): %q", got)
	}
	if got := c.WithName("other"); got.Name != "other" {
		t.Fatalf("unexpected Color.WithName(): %+v", got)
	}
	if got := c.Apply("x"); !strings.Contains(got, "x") {
		t.Fatalf("unexpected Color.Apply(): %q", got)
	}
	if got := c.ApplyMode("x", ModeNoColor); got != "x" {
		t.Fatalf("unexpected Color.ApplyMode(): %q", got)
	}
	if got := RGB(1, 2, 3); got.R != 1 {
		t.Fatalf("unexpected RGB() helper: %+v", got)
	}
	if got := Hex("#010203"); got.Hex != "#010203" {
		t.Fatalf("unexpected Hex() helper: %+v", got)
	}
	if got := ANSIFunc(2); got.ANSI != 2 {
		t.Fatalf("unexpected ANSIFunc() helper: %+v", got)
	}
	if got := Color256Func(2); got.Color256 != 2 {
		t.Fatalf("unexpected Color256Func() helper: %+v", got)
	}
	if got := DefaultColorConfig().Mode.String(); got != "TrueColor" {
		t.Fatalf("unexpected Mode.String(): %q", got)
	}
	if got := Mode(99).String(); got != "Unknown" {
		t.Fatalf("unexpected unknown mode string: %q", got)
	}
}
