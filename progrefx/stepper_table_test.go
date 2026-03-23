package progrefx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oxhq/tfx/internal/share"
)

func TestStepperDefaultsAndAdvance(t *testing.T) {
	t.Parallel()

	stepper := Stepper(StepperConfig{
		Label: "Release",
		Steps: []StepItem{
			{Title: "Boot"},
			{Title: "Ship"},
		},
		Writer:    &bytes.Buffer{},
		DetectTTY: nonTTY(),
	})

	rendered := stepper.Render()
	if !strings.Contains(rendered, "[now ] Boot") {
		t.Fatalf("expected first step to be active by default, got %q", rendered)
	}

	stepper.Advance()
	rendered = stepper.Render()
	if !strings.Contains(rendered, "[done] Boot") || !strings.Contains(rendered, "[now ] Ship") {
		t.Fatalf("expected advance to complete first step and activate second, got %q", rendered)
	}

	stepper.Advance()
	rendered = stepper.Render()
	if !strings.Contains(rendered, "[done] Ship") {
		t.Fatalf("expected final step to be completed, got %q", rendered)
	}
}

func TestStepperBuilderAndMutations(t *testing.T) {
	t.Parallel()

	stepper := NewStepperBuilder().
		Label("Pipeline").
		Theme(GitHubTheme).
		DetectTTY(ttyANSI()).
		Steps([]StepItem{
			{Title: "Compile"},
			{Title: "Verify"},
		}).
		Build()

	if got := stepper.Steps(); len(got) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(got))
	}

	stepper.SetDetail(0, "running")
	stepper.Complete(0)
	stepper.Fail(1, "lint failed")
	rendered := stripANSI(stepper.Render())
	if !strings.Contains(rendered, "Compile - running") {
		t.Fatalf("expected detail to be rendered, got %q", rendered)
	}
	if !strings.Contains(rendered, "Verify - lint failed") {
		t.Fatalf("expected failure detail to be rendered, got %q", rendered)
	}
}

func TestTryStepperRejectsInvalidMultipathArgs(t *testing.T) {
	t.Parallel()

	if _, err := TryStepper("bad"); err == nil {
		t.Fatal("expected invalid stepper config error")
	}

	if _, err := TryStepper(share.Option[StepperConfig](func(cfg *StepperConfig) {
		cfg.Label = "ok"
	}), "bad"); err == nil {
		t.Fatal("expected invalid mixed stepper config error")
	}
}

func TestTableRenderAndBuilder(t *testing.T) {
	t.Parallel()

	table := Table(TableConfig{
		Title: "Coverage",
		Columns: []TableColumn{
			{Title: "Pkg"},
			{Title: "Pct", Align: TableAlignRight},
		},
		Rows:      [][]string{{"progrefx", "72.7%"}},
		Writer:    &bytes.Buffer{},
		DetectTTY: nonTTY(),
	})

	rendered := table.Render()
	if !strings.Contains(rendered, "Coverage") || !strings.Contains(rendered, "progrefx") {
		t.Fatalf("expected title and row in table render, got %q", rendered)
	}

	built := NewTableBuilder().
		Title("Packages").
		Columns([]TableColumn{{Title: "Name"}, {Title: "Status"}}).
		Rows([][]string{{"flowfx", "ok"}}).
		Theme(DraculaTheme).
		Style(TableStyleASCII).
		DetectTTY(nonTTY()).
		Build()

	built.AppendRow([]string{"runfx", "ok"})
	rows := built.Rows()
	if len(rows) != 2 {
		t.Fatalf("expected appended row, got %d rows", len(rows))
	}

	rendered = built.Render()
	if !strings.Contains(rendered, "+") || !strings.Contains(rendered, "runfx") {
		t.Fatalf("expected ascii table render, got %q", rendered)
	}
}

func TestTryTableRejectsInvalidMultipathArgs(t *testing.T) {
	t.Parallel()

	if _, err := TryTable("bad"); err == nil {
		t.Fatal("expected invalid table config error")
	}
}
