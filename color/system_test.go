package color

import "testing"

func TestDefaultThemeSwitching(t *testing.T) {
	t.Parallel()

	originalTheme := GetDefaultTheme()
	originalEncoding := GetDefaultEncoding()
	t.Cleanup(func() {
		SetDefaultTheme(originalTheme)
		SetDefaultEncoding(originalEncoding)
	})

	SetDefaultTheme("nord")
	if got := GetDefaultTheme(); got != "nord" {
		t.Fatalf("expected nord theme, got %q", got)
	}
	if Blue != Nord.Blue {
		t.Fatalf("expected nord blue to become default blue, got %+v", Blue)
	}

	UseGitHub()
	if got := GetDefaultTheme(); got != "github" {
		t.Fatalf("expected github theme, got %q", got)
	}
	if Green != GitHub.Green {
		t.Fatalf("expected github green to become default green, got %+v", Green)
	}

	SetDefaultEncoding(Mode256Color)
	if got := GetDefaultEncoding(); got != Mode256Color {
		t.Fatalf("expected 256-color mode, got %v", got)
	}

	UseMaterial()
	UseDracula()
	UseNord()
}
