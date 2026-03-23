package progrefx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/terminal"
)

func forcedDetector(mode terminal.Mode) *terminal.Detector {
	detector := terminal.NewDetector(&bytes.Buffer{})
	detector.ForceMode(mode)
	return detector
}

func TestProgressBuilderEffectsAndAdd(t *testing.T) {
	t.Parallel()

	progress := NewProgressBuilder().
		Total(5).
		Label("Coverage").
		Width(3).
		Theme(NeonTheme).
		Style(ProgressStyleDots).
		Effect(EffectGlow).
		ShowETA(true).
		Writer(&bytes.Buffer{}).
		DetectTTY(nonTTY()).
		Build()

	if progress.total != 5 || progress.label != "Coverage" || progress.width != 3 {
		t.Fatalf("unexpected progress builder state: %+v", progress)
	}
	if progress.theme != NeonTheme || progress.style != ProgressStyleDots ||
		progress.effect != EffectGlow || !progress.ShowETA {
		t.Fatalf("expected builder options to propagate, got %+v", progress)
	}

	progress.Add(2)
	progress.Add(10)
	if !progress.isStarted || progress.current != 5 {
		t.Fatalf(
			"expected Add to start and cap progress, got started=%v current=%d",
			progress.isStarted,
			progress.current,
		)
	}
	if got := progress.Render(); got != "Coverage 100%" {
		t.Fatalf("expected non-TTY render after Add, got %q", got)
	}
}

func TestStepperAndTableSetters(t *testing.T) {
	t.Parallel()

	stepper := NewStepperBuilder().
		Label("Original").
		Steps([]StepItem{{Title: "One"}, {Title: "Two"}}).
		Theme(GitHubTheme).
		Writer(&bytes.Buffer{}).
		DetectTTY(nonTTY()).
		Build()

	stepper.SetLabel("Updated")
	stepper.SetSteps([]StepItem{{Title: "A"}, {Title: "B", Status: StepPending}})
	stepper.Activate(1)
	stepper.Skip(0)
	steps := stepper.Steps()
	if stepper.label != "Updated" {
		t.Fatalf("expected label update, got %q", stepper.label)
	}
	if len(steps) != 2 || steps[0].Status != StepSkipped || steps[1].Status != StepActive {
		t.Fatalf("unexpected stepper statuses after setters: %#v", steps)
	}

	table := NewTableBuilder().
		Title("Before").
		Columns([]TableColumn{{Title: "Name"}}).
		Rows([][]string{{"old"}}).
		Theme(NordTheme).
		Style(TableStyleUnicode).
		Writer(&bytes.Buffer{}).
		DetectTTY(nonTTY()).
		Build()

	table.SetTitle("After")
	table.SetRows([][]string{{"new"}, {"next"}})
	rows := table.Rows()
	rows[0][0] = "mutated"

	if table.title != "After" {
		t.Fatalf("expected title update, got %q", table.title)
	}
	gotRows := table.Rows()
	if len(gotRows) != 2 || gotRows[0][0] != "new" {
		t.Fatalf("expected SetRows to replace and Rows() to clone, got %#v", gotRows)
	}
}

func TestThemeEffectsAndHelpers(t *testing.T) {
	t.Parallel()

	detector := forcedDetector(terminal.ModeANSI)

	rainbow := NeonTheme.RenderProgress(0.5, 4, EffectRainbow, ProgressStyleAscii, detector)
	if !strings.Contains(rainbow, color.Reset) || !strings.Contains(rainbow, "=") {
		t.Fatalf("expected rainbow progress to render ANSI colored bar, got %q", rainbow)
	}

	glow := NeonTheme.RenderProgress(0.5, 4, EffectGlow, ProgressStyleAscii, detector)
	if !strings.Contains(glow, color.Reset) || !strings.Contains(glow, "=") {
		t.Fatalf("expected glow progress to render ANSI colored bar, got %q", glow)
	}

	if got := minInt(1, 2); got != 1 {
		t.Fatalf("unexpected minInt result: %d", got)
	}
	if got := minInt(3, 2); got != 2 {
		t.Fatalf("unexpected minInt result: %d", got)
	}

	if got := intToUint8(-1); got != 0 {
		t.Fatalf("expected negative values to clamp to 0, got %d", got)
	}
	if got := intToUint8(999); got != 255 {
		t.Fatalf("expected large values to clamp to 255, got %d", got)
	}
	if got := intToUint8(42); got != 42 {
		t.Fatalf("expected in-range values to pass through, got %d", got)
	}
}
