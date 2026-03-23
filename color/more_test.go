package color

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oxhq/tfx/terminal"
)

func ansiDetector() *terminal.Detector {
	detector := terminal.NewDetector(&bytes.Buffer{})
	detector.ForceMode(terminal.ModeANSI)
	return detector
}

func TestColorEffectsAndPaletteFactories(t *testing.T) {
	t.Parallel()

	detector := ansiDetector()

	if got := RainbowString("", detector); got != "" {
		t.Fatalf("expected empty rainbow string, got %q", got)
	}
	if got := RainbowString("abc", detector); !strings.Contains(got, "a") ||
		!strings.Contains(got, Reset) {
		t.Fatalf("unexpected rainbow string: %q", got)
	}

	if got := WaveString("", ColorBlue, 20, detector); got != "" {
		t.Fatalf("expected empty wave string, got %q", got)
	}
	if got := WaveString("abc", ColorBlue, 20, detector); !strings.Contains(got, "a") ||
		!strings.Contains(got, Reset) {
		t.Fatalf("unexpected wave string: %q", got)
	}

	if got := GradientString("ab", ColorRed, ColorBlue, detector); !strings.Contains(got, "a") ||
		!strings.Contains(got, "b") || !strings.Contains(got, Reset) {
		t.Fatalf("unexpected gradient string: %q", got)
	}

	palettes := map[string]Palette{
		"dracula":  DraculaPalette(),
		"nord":     NordPalette(),
		"tailwind": TailwindPalette(),
		"github":   GitHubPalette(),
		"vscode":   VSCodePalette(),
	}
	for name, palette := range palettes {
		if len(palette) == 0 {
			t.Fatalf("expected palette %s to contain colors", name)
		}
	}
	if _, ok := palettes["dracula"].Get("purple"); !ok {
		t.Fatal("expected dracula palette to expose purple")
	}
	if _, ok := palettes["nord"].Get("cyan"); !ok {
		t.Fatal("expected nord palette to expose cyan")
	}
	if _, ok := palettes["tailwind"].Get("emerald"); !ok {
		t.Fatal("expected tailwind palette to expose emerald")
	}
	if _, ok := palettes["github"].Get("blue_light"); !ok {
		t.Fatal("expected github palette to expose blue_light")
	}
	if _, ok := palettes["vscode"].Get("orange"); !ok {
		t.Fatal("expected vscode palette to expose orange")
	}
}

func TestColorConversionAndFormattingHelpers(t *testing.T) {
	t.Parallel()

	shortHex := NewHex("#abc")
	if shortHex.Hex != "#AABBCC" || shortHex.R != 0xAA || shortHex.G != 0xBB || shortHex.B != 0xCC {
		t.Fatalf("expected short hex expansion, got %+v", shortHex)
	}

	if got := (Color{Hex: "#010203"}).String(); got != "#010203" {
		t.Fatalf("expected unnamed color to stringify to hex, got %q", got)
	}

	if got := (Color{ANSI: 9}).Render(ModeANSI); got != "\033[91m" {
		t.Fatalf("unexpected bright ANSI render: %q", got)
	}
	if got := (Color{ANSI: 9, IsBg: true}).Background(ModeANSI); got != "\033[101m" {
		t.Fatalf("unexpected bright ANSI background: %q", got)
	}
	if got := (Color{ANSI: 99}).Render(ModeANSI); got != "" {
		t.Fatalf("expected unsupported ANSI code to render empty string, got %q", got)
	}
	if got := (Color{ANSI: 99, IsBg: true}).Background(ModeANSI); got != "" {
		t.Fatalf("expected unsupported ANSI background to render empty string, got %q", got)
	}

	if r, g, b := ansiToRGB(10); r != 0 || g != 255 || b != 0 {
		t.Fatalf("unexpected ansiToRGB result: %d %d %d", r, g, b)
	}
	if r, g, b := ansiToRGB(99); r != 0 || g != 0 || b != 0 {
		t.Fatalf("expected out-of-range ansiToRGB to return zeros, got %d %d %d", r, g, b)
	}

	if r, g, b := color256ToRGB(17); r != 0 || g != 0 || b != 51 {
		t.Fatalf("unexpected cube color conversion: %d %d %d", r, g, b)
	}
	if r, g, b := color256ToRGB(240); r != 88 || g != 88 || b != 88 {
		t.Fatalf("unexpected grayscale conversion: %d %d %d", r, g, b)
	}

	if got := clampUint8(-5); got != 0 {
		t.Fatalf("expected negative clamp to 0, got %d", got)
	}
	if got := clampUint8(999); got != 255 {
		t.Fatalf("expected high clamp to 255, got %d", got)
	}
	if got := clampUint8(42); got != 42 {
		t.Fatalf("expected in-range clamp to pass through, got %d", got)
	}
}
