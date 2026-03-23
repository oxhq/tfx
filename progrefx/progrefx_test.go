package progrefx

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
	"github.com/oxhq/tfx/terminal"
)

func nonTTY() func() runfx.TTYInfo {
	return func() runfx.TTYInfo {
		return runfx.TTYInfo{IsTTY: false}
	}
}

func ttyANSI() func() runfx.TTYInfo {
	return func() runfx.TTYInfo {
		return runfx.TTYInfo{IsTTY: true}
	}
}

func forceProgressTTY(p *Progress, mode terminal.Mode) {
	p.isTTY = true
	if p.detector != nil {
		p.detector.ForceMode(mode)
	}
}

func forceSpinnerTTY(s *Spinner, mode terminal.Mode) {
	s.isTTY = true
	if s.detector != nil {
		s.detector.ForceMode(mode)
	}
}

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

func TestStartAndProgressBuilderNonTTY(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	progress := Start(
		ProgressConfig{
			Total:     12,
			Label:     "Ignored",
			Width:     6,
			Writer:    buf,
			DetectTTY: nonTTY(),
		},
		share.Option[ProgressConfig](func(cfg *ProgressConfig) {
			cfg.Label = "Configured"
			cfg.Total = 20
		}),
	)

	if progress.total != 20 {
		t.Fatalf("expected option to override total, got %d", progress.total)
	}
	if progress.label != "Configured" {
		t.Fatalf("expected option to override label, got %q", progress.label)
	}
	if got := progress.Render(); got != "" {
		t.Fatalf("expected empty render before start, got %q", got)
	}

	progress.Set(10)
	if got := progress.Render(); got != "Configured  50%" {
		t.Fatalf("expected non-TTY render, got %q", got)
	}

	progress.SetLabel("Updated")
	progress.Finish()
	if progress.current != 20 {
		t.Fatalf("expected Finish to set total, got %d", progress.current)
	}
	if got := progress.Render(); got != "Updated 100%" {
		t.Fatalf("expected finished render, got %q", got)
	}

	built := NewProgressBuilder().
		Total(7).
		Label("Builder").
		Width(3).
		Theme(GitHubTheme).
		Style(ProgressStyleAscii).
		DetectTTY(nonTTY()).
		Build()

	if built.total != 7 || built.label != "Builder" || built.width != 3 {
		t.Fatalf("builder did not propagate config: %+v", built)
	}
	if built.theme != GitHubTheme {
		t.Fatalf("expected builder theme to propagate, got %+v", built.theme)
	}
	if built.style != ProgressStyleAscii {
		t.Fatalf("expected builder style to propagate, got %v", built.style)
	}

	built.Set(7)
	if got := built.Render(); got != "Builder 100%" {
		t.Fatalf("expected builder render, got %q", got)
	}
}

func TestTryStartRejectsInvalidMultipathArgs(t *testing.T) {
	t.Parallel()

	if _, err := TryStart("bad"); err == nil {
		t.Fatal("expected invalid config error")
	}

	if _, err := TryStartSpinner("bad"); err == nil {
		t.Fatal("expected invalid spinner config error")
	}
}

func TestRenderBarTTYWithEffectAndETA(t *testing.T) {
	t.Parallel()

	progress := Start(
		ProgressConfig{
			Total:     100,
			Label:     "Download",
			Width:     8,
			Theme:     DraculaTheme,
			Effect:    EffectGradient,
			ShowETA:   true,
			Writer:    &bytes.Buffer{},
			DetectTTY: ttyANSI(),
		},
	)

	progress.Set(25)
	progress.startTime = time.Now().Add(-5 * time.Second)
	forceProgressTTY(progress, terminal.ModeANSI)

	output := RenderBar(progress, progress.detector)
	if !strings.HasPrefix(output, "\r") {
		t.Fatalf("expected TTY output to start with carriage return, got %q", output)
	}
	if !strings.Contains(output, "Download") {
		t.Fatalf("expected label in output, got %q", output)
	}
	if !strings.Contains(output, "25%") {
		t.Fatalf("expected percentage in output, got %q", output)
	}
	if !strings.Contains(output, "ETA:") {
		t.Fatalf("expected ETA in output, got %q", output)
	}
	if !strings.Contains(output, color.Reset) {
		t.Fatalf("expected ANSI reset sequences in TTY output, got %q", output)
	}
}

func TestRenderBarUsesConfiguredStyle(t *testing.T) {
	t.Parallel()

	progress, err := TryStart(ProgressConfig{
		Total:     4,
		Label:     "Styled",
		Width:     4,
		Theme:     MaterialTheme,
		Style:     ProgressStyleAscii,
		Writer:    &bytes.Buffer{},
		DetectTTY: ttyANSI(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	progress.Set(2)
	forceProgressTTY(progress, terminal.ModeANSI)

	output := stripANSI(RenderBar(progress, progress.detector))
	if !strings.Contains(output, "==--") {
		t.Fatalf("expected ascii style render, got %q", output)
	}
}

func TestThemeFromPaletteAndRenderColor(t *testing.T) {
	t.Parallel()

	palette := color.Palette{
		"green": color.NewRGB(1, 2, 3),
		"gray":  color.NewRGB(4, 5, 6),
		"blue":  color.NewRGB(7, 8, 9),
	}

	theme := NewThemeFromPalette("custom", palette)
	if theme.Name != "custom" {
		t.Fatalf("expected theme name to be preserved, got %q", theme.Name)
	}
	if theme.CompleteColor != palette["green"] {
		t.Fatalf("expected green to map to CompleteColor, got %+v", theme.CompleteColor)
	}
	if theme.IncompleteColor != palette["gray"] {
		t.Fatalf("expected gray to map to IncompleteColor, got %+v", theme.IncompleteColor)
	}
	if theme.LabelColor != palette["blue"] {
		t.Fatalf("expected blue to map to LabelColor, got %+v", theme.LabelColor)
	}
	if theme.PercentColor != palette["blue"] {
		t.Fatalf("expected percent color to match label color, got %+v", theme.PercentColor)
	}
	if theme.EffectEnabled {
		t.Fatal("expected palette themes to disable effects by default")
	}

	got := theme.RenderColor(theme.LabelColor, nil)
	want := theme.LabelColor.Render(color.ModeANSI)
	if got != want {
		t.Fatalf("expected nil detector to use ANSI mode, got %q want %q", got, want)
	}
}

func TestStartSpinnerBuilderAndTick(t *testing.T) {
	t.Parallel()

	spinner := StartSpinner(
		SpinnerConfig{
			Label:     "Loading",
			Frames:    []string{"a", "b"},
			Theme:     MaterialTheme,
			Writer:    &bytes.Buffer{},
			DetectTTY: nonTTY(),
		},
	)

	if got := spinner.Render(); got != "Loading" {
		t.Fatalf("expected non-TTY spinner to return label, got %q", got)
	}
	spinner.Tick()
	spinner.SetLabel("Working")
	if got := spinner.Render(); got != "Working" {
		t.Fatalf("expected non-TTY spinner to keep returning label, got %q", got)
	}

	built := NewSpinnerBuilder().
		Label("Spin").
		Frames([]string{"x", "y"}).
		Theme(DraculaTheme).
		DetectTTY(ttyANSI()).
		Build()
	forceSpinnerTTY(built, terminal.ModeANSI)

	first := built.Render()
	if !strings.Contains(first, "x") || !strings.Contains(first, "Spin") {
		t.Fatalf("expected first frame and label, got %q", first)
	}
	if !strings.Contains(first, color.Reset) {
		t.Fatalf("expected ANSI reset sequences in TTY spinner output, got %q", first)
	}

	built.Tick()
	built.SetLabel("Updated")
	second := built.Render()
	if !strings.Contains(second, "y") || !strings.Contains(second, "Updated") {
		t.Fatalf("expected ticked frame and updated label, got %q", second)
	}
}
