package color

import (
	"strings"
	"testing"
)

func TestGradientStringSingleRune(t *testing.T) {
	from := NewRGB(255, 0, 0)
	to := NewRGB(0, 0, 255)

	got := GradientString("A", from, to, nil)
	want := from.Render(ModeANSI) + "A" + Reset

	if got != want {
		t.Fatalf("unexpected gradient for single rune:\nwant %q\ngot  %q", want, got)
	}
}

func TestGlitchStringPreservesInputRunes(t *testing.T) {
	got := GlitchString("ab", nil)

	if got == "" {
		t.Fatal("expected glitch string to be non-empty")
	}
	if !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Fatalf("expected output to contain both input runes, got %q", got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI escapes in glitch output, got %q", got)
	}
}
