package formfx

import (
	"errors"
	"strings"
	"testing"

	"github.com/oxhq/tfx/runfx"
)

func TestTextInputKeyHandlerProcessesEditingKeys(t *testing.T) {
	prompt := NewInputPrompt("")

	if stop := prompt.OnKey(runfx.Key{Rune: 'a'}); stop {
		t.Fatal("expected typing to continue")
	}
	if stop := prompt.OnKey(runfx.Key{Rune: 'b'}); stop {
		t.Fatal("expected typing to continue")
	}
	if got := string(prompt.Value); got != "ab" {
		t.Fatalf("expected ab, got %q", got)
	}

	prompt.CursorPos = 1
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyArrowLeft}); stop {
		t.Fatal("expected arrow left to continue")
	}
	if prompt.CursorPos != 0 {
		t.Fatalf("expected cursor at 0, got %d", prompt.CursorPos)
	}

	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyBackspace}); stop {
		t.Fatal("expected backspace to continue")
	}
	if got := string(prompt.Value); got != "ab" {
		t.Fatalf("expected unchanged value at start of buffer, got %q", got)
	}

	prompt.CursorPos = 1
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyBackspace}); stop {
		t.Fatal("expected backspace to continue")
	}
	if got := string(prompt.Value); got != "b" {
		t.Fatalf("expected b after backspace, got %q", got)
	}
	if prompt.CursorPos != 0 {
		t.Fatalf("expected cursor at 0, got %d", prompt.CursorPos)
	}

	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyArrowRight}); stop {
		t.Fatal("expected arrow right to continue")
	}
	if prompt.CursorPos != 1 {
		t.Fatalf("expected cursor at 1, got %d", prompt.CursorPos)
	}

	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyEnter}); !stop {
		t.Fatal("expected enter to stop")
	}
	select {
	case got := <-prompt.Done:
		if got != "b" {
			t.Fatalf("expected done value b, got %q", got)
		}
	default:
		t.Fatal("expected done value")
	}

	other := NewInputPrompt("")
	if stop := other.OnKey(runfx.Key{Code: runfx.KeyEscape}); !stop {
		t.Fatal("expected escape to stop")
	}
	select {
	case _, ok := <-other.Canceled:
		if ok {
			t.Fatal("expected canceled channel to be closed")
		}
	default:
		t.Fatal("expected canceled channel to be closed")
	}
}

func TestSelectPromptSetOptionsPreservesCustomHandler(t *testing.T) {
	custom := func(p *Prompt, key runfx.Key) bool {
		if key.Rune == 'n' && p.NumOptions > 0 {
			p.SelectedIndex = (p.SelectedIndex + 1) % p.NumOptions
		}
		return false
	}

	prompt, err := NewSelectPrompt(SelectConfig{
		Label:         "Choose",
		Options:       []string{"one", "two"},
		KeyHandler:    custom,
		SelectedIndex: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := prompt.SetOptions([]string{"one", "two", "three"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stop := prompt.OnKey(runfx.Key{Rune: 'n'}); stop {
		t.Fatal("expected custom handler to continue")
	}
	if got := prompt.prompt.SelectedIndex; got != 1 {
		t.Fatalf("expected custom handler to survive SetOptions, got %d", got)
	}
}

func TestSecretPromptConfirmFlow(t *testing.T) {
	prompt, err := NewSecretPrompt(SecretConfig{
		Label:   "Secret",
		Confirm: true,
		Mask:    '#',
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range "abc" {
		if stop := prompt.OnKey(runfx.Key{Rune: r}); stop {
			t.Fatal("expected typing to continue")
		}
	}
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyEnter}); stop {
		t.Fatal("expected first enter to switch to confirmation")
	}
	if !prompt.confirming {
		t.Fatal("expected confirm state to be enabled")
	}
	if got := string(prompt.prompt.Value); got != "" {
		t.Fatalf("expected cleared buffer, got %q", got)
	}
	rendered := string(prompt.Render())
	if !strings.Contains(rendered, "(confirm)") {
		t.Fatalf("expected confirm label in render, got %q", rendered)
	}

	for _, r := range "abc" {
		if stop := prompt.OnKey(runfx.Key{Rune: r}); stop {
			t.Fatal("expected typing to continue")
		}
	}
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyEnter}); !stop {
		t.Fatal("expected confirmation enter to stop")
	}

	select {
	case got := <-prompt.Done():
		if got != "abc" {
			t.Fatalf("expected abc, got %q", got)
		}
	default:
		t.Fatal("expected done value")
	}
}

func TestPromptSetKeyHandlerNilKeepsExistingHandler(t *testing.T) {
	prompt, err := NewPrompt(2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prompt.SetKeyHandler(nil)
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyArrowRight}); stop {
		t.Fatal("expected prompt to continue")
	}
	if prompt.SelectedIndex != 1 {
		t.Fatalf("expected existing handler to remain active, got index %d", prompt.SelectedIndex)
	}
}

func TestConfirmRejectsInvalidMultipathArgs(t *testing.T) {
	if _, err := Confirm("bad"); err == nil {
		t.Fatal("expected invalid config error")
	} else if !errors.Is(err, ErrInvalidConfigType) {
		t.Fatalf("expected ErrInvalidConfigType, got %v", err)
	}
}

func TestSelectRejectsInvalidMultipathArgs(t *testing.T) {
	if _, err := Select(123); err == nil {
		t.Fatal("expected invalid config error")
	} else if !errors.Is(err, ErrInvalidConfigType) {
		t.Fatalf("expected ErrInvalidConfigType, got %v", err)
	}
}

func TestSecretRejectsInvalidMultipathArgs(t *testing.T) {
	if _, err := Secret(map[string]string{"bad": "config"}); err == nil {
		t.Fatal("expected invalid config error")
	} else if !errors.Is(err, ErrInvalidConfigType) {
		t.Fatalf("expected ErrInvalidConfigType, got %v", err)
	}
}

func TestConfirmBuilderCollectsValidationErrors(t *testing.T) {
	builder := NewConfirmBuilder("valid", true).KeyHandler(nil)
	if _, err := builder.Build(); err == nil {
		t.Fatal("expected invalid key handler error")
	} else if !errors.Is(err, ErrInvalidKeyHandler) {
		t.Fatalf("expected ErrInvalidKeyHandler, got %v", err)
	}

	builder = NewConfirmBuilder("valid", true).Renderer(nil)
	if _, err := builder.Build(); err == nil {
		t.Fatal("expected invalid renderer error")
	} else if !errors.Is(err, ErrInvalidRenderer) {
		t.Fatalf("expected ErrInvalidRenderer, got %v", err)
	}

	builder = NewConfirmBuilder("valid", true).Label("")
	if _, err := builder.Build(); err == nil {
		t.Fatal("expected invalid label error")
	} else if !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected ErrInvalidOption, got %v", err)
	}
}
