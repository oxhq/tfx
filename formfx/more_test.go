package formfx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/oxhq/tfx/runfx"
)

type confirmRendererStub struct {
	out []byte
}

func (r confirmRendererStub) Render(*ConfirmPrompt) []byte { return r.out }

type selectRendererStub struct {
	out []byte
}

func (r selectRendererStub) Render(*SelectPrompt) []byte { return r.out }

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestConfirmPromptDefaultsAndMethods(t *testing.T) {
	if _, err := NewConfirmPrompt(nil); err == nil || !errors.Is(err, ErrConfigNotSet) {
		t.Fatalf("expected ErrConfigNotSet, got %v", err)
	}

	prompt, err := NewConfirmBuilder("Delete file?", true).
		DefaultValue(false).
		Label("Delete file?").
		Build()
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if prompt.prompt.SelectedIndex != 1 {
		t.Fatalf("expected default selection to point to No, got %d", prompt.prompt.SelectedIndex)
	}
	if prompt.Done() != prompt.prompt.Done {
		t.Fatal("expected done channel passthrough")
	}
	if prompt.Canceled() != prompt.prompt.Canceled {
		t.Fatal("expected canceled channel passthrough")
	}

	rendered := string(prompt.Render())
	if !strings.Contains(rendered, "Delete file?") || !strings.Contains(rendered, "[No]") {
		t.Fatalf("unexpected confirm render: %q", rendered)
	}

	prompt.SetRenderer(confirmRendererStub{out: []byte("custom confirm")})
	if got := string(prompt.Render()); got != "custom confirm" {
		t.Fatalf("expected custom renderer output, got %q", got)
	}

	prompt.SetRenderer(&DefaultConfirmRenderer{})
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyArrowLeft}); stop {
		t.Fatal("expected navigation to continue")
	}
	if prompt.prompt.SelectedIndex != 0 {
		t.Fatalf("expected selection to move to Yes, got %d", prompt.prompt.SelectedIndex)
	}
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyEnter}); !stop {
		t.Fatal("expected enter to stop")
	}

	select {
	case got := <-prompt.Done():
		if got != 0 {
			t.Fatalf("expected selected index 0, got %d", got)
		}
	default:
		t.Fatal("expected confirm result on done channel")
	}

	prompt, err = NewConfirmPrompt(&ConfirmConfig{Label: "Cancel?", DefaultValue: true})
	if err != nil {
		t.Fatalf("unexpected config error: %v", err)
	}
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyEscape}); !stop {
		t.Fatal("expected escape to stop")
	}
	select {
	case _, ok := <-prompt.Canceled():
		if ok {
			t.Fatal("expected canceled channel to be closed")
		}
	default:
		t.Fatal("expected canceled channel to be closed")
	}

	prompt.OnResize(120, 40)
}

func TestSelectPromptBuilderAndHelpers(t *testing.T) {
	if _, err := NewSelectPrompt(SelectConfig{}); err == nil {
		t.Fatal("expected invalid select config error")
	}

	prompt, err := NewSelectBuilder().
		Label("Pick one").
		Options([]string{"one", "two"}).
		SelectedIndex(99).
		Build()
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if prompt.prompt.SelectedIndex != 1 {
		t.Fatalf("expected selected index to clamp to 1, got %d", prompt.prompt.SelectedIndex)
	}
	if prompt.Done() != prompt.prompt.Done {
		t.Fatal("expected done channel passthrough")
	}
	if prompt.Canceled() != prompt.prompt.Canceled {
		t.Fatal("expected canceled channel passthrough")
	}

	rendered := string(prompt.Render())
	if !strings.Contains(rendered, "Pick one") || !strings.Contains(rendered, "> two") {
		t.Fatalf("unexpected select render: %q", rendered)
	}

	prompt.SetRenderer(selectRendererStub{out: []byte("custom select")})
	if got := string(prompt.Render()); got != "custom select" {
		t.Fatalf("expected custom renderer output, got %q", got)
	}

	prompt.SetRenderer(&DefaultSelectRenderer{})
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyArrowUp}); stop {
		t.Fatal("expected navigation to continue")
	}
	if prompt.prompt.SelectedIndex != 0 {
		t.Fatalf("expected selection to move up, got %d", prompt.prompt.SelectedIndex)
	}
	if stop := prompt.OnKey(runfx.Key{Code: runfx.KeyEnter}); !stop {
		t.Fatal("expected enter to stop")
	}

	select {
	case got := <-prompt.Done():
		if got != 0 {
			t.Fatalf("expected selected index 0, got %d", got)
		}
	default:
		t.Fatal("expected selected index on done channel")
	}

	if err := prompt.SetOptions([]string{}); err == nil {
		t.Fatal("expected empty options error")
	}
	if err := prompt.SetOptions([]string{"solo"}); err != nil {
		t.Fatalf("unexpected set options error: %v", err)
	}
	if prompt.prompt.SelectedIndex != 0 {
		t.Fatalf("expected selected index to clamp to 0, got %d", prompt.prompt.SelectedIndex)
	}

	cancelPrompt, err := Select(SelectConfig{
		Label:   "Cancel select",
		Options: []string{"one", "two"},
	})
	if err != nil {
		t.Fatalf("unexpected select creation error: %v", err)
	}
	if stop := cancelPrompt.OnKey(runfx.Key{Code: runfx.KeyEscape}); !stop {
		t.Fatal("expected escape to stop")
	}
	select {
	case _, ok := <-cancelPrompt.Canceled():
		if ok {
			t.Fatal("expected canceled channel to be closed")
		}
	default:
		t.Fatal("expected canceled channel to be closed")
	}

	prompt.OnResize(80, 24)
}

func TestPromptInputAndIOHelpers(t *testing.T) {
	prompt, err := NewPrompt(1, 10)
	if err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}
	if prompt.SelectedIndex != 0 {
		t.Fatalf("expected clamped selected index 0, got %d", prompt.SelectedIndex)
	}
	if prompt.Render() != nil {
		t.Fatalf("expected nil render output, got %v", prompt.Render())
	}
	prompt.OnResize(100, 30)

	prompt = &Prompt{
		NumOptions:    3,
		SelectedIndex: 0,
		Done:          make(chan int, 1),
		Canceled:      make(chan struct{}),
	}
	if stop := HorizontalKeyHandler(prompt, runfx.Key{Code: runfx.KeyTab}); stop {
		t.Fatal("expected tab to continue")
	}
	if prompt.SelectedIndex != 1 {
		t.Fatalf("expected tab to cycle to 1, got %d", prompt.SelectedIndex)
	}

	vertical := &Prompt{
		NumOptions:    2,
		SelectedIndex: 0,
		Done:          make(chan int, 1),
		Canceled:      make(chan struct{}),
	}
	if stop := VerticalKeyHandler(vertical, runfx.Key{Code: runfx.KeyArrowDown}); stop {
		t.Fatal("expected arrow down to continue")
	}
	if vertical.SelectedIndex != 1 {
		t.Fatalf("expected arrow down to move selection, got %d", vertical.SelectedIndex)
	}
	if stop := VerticalKeyHandler(vertical, runfx.Key{Code: runfx.KeyEnter}); !stop {
		t.Fatal("expected enter to stop")
	}
	select {
	case got := <-vertical.Done:
		if got != 1 {
			t.Fatalf("expected selected index 1, got %d", got)
		}
	default:
		t.Fatal("expected done selection")
	}

	customPrompt, err := NewPrompt(2, 0)
	if err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}
	customCalled := false
	customPrompt.SetKeyHandler(func(p *Prompt, key runfx.Key) bool {
		customCalled = true
		p.SelectedIndex = 1
		return true
	})
	if stop := customPrompt.OnKey(runfx.Key{Rune: 'x'}); !stop {
		t.Fatal("expected custom handler to stop")
	}
	if !customCalled || customPrompt.SelectedIndex != 1 {
		t.Fatal("expected custom key handler to be used")
	}

	input := NewInputPrompt("x")
	input.SetKeyHandler(nil)
	if stop := input.OnKey(runfx.Key{Code: runfx.KeySpace}); stop {
		t.Fatal("expected input fallback handler to continue")
	}
	if got := string(input.Value); got != "x " {
		t.Fatalf("expected fallback handler to insert space, got %q", got)
	}
	if input.Render() != nil {
		t.Fatalf("expected nil input render, got %v", input.Render())
	}
	input.OnResize(80, 24)

	customInputCalled := false
	input.SetKeyHandler(func(p *InputPrompt, key runfx.Key) bool {
		customInputCalled = true
		p.CursorPos = 0
		return true
	})
	if stop := input.OnKey(runfx.Key{Rune: 'z'}); !stop {
		t.Fatal("expected custom input handler to stop")
	}
	if !customInputCalled || input.CursorPos != 0 {
		t.Fatal("expected custom input handler to run")
	}

	stdinReader := NewStdinReader(strings.NewReader("hello world"))
	line, err := stdinReader.ReadLine(context.Background())
	if err != nil {
		t.Fatalf("unexpected read line error: %v", err)
	}
	if line != "hello world" {
		t.Fatalf("expected full reader contents, got %q", line)
	}

	wantErr := io.EOF
	if _, err := NewStdinReader(failingReader{err: wantErr}).
		ReadLine(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected read error %v, got %v", wantErr, err)
	}

	var buf bytes.Buffer
	stdoutWriter := NewStdoutWriter(&buf)
	if n, err := stdoutWriter.Write([]byte("abc")); err != nil || n != 3 {
		t.Fatalf("unexpected Write result: n=%d err=%v", n, err)
	}
	if n, err := stdoutWriter.WriteString("123"); err != nil || n != 3 {
		t.Fatalf("unexpected WriteString result: n=%d err=%v", n, err)
	}
	stdoutWriter.Flush()
	if got := buf.String(); got != "abc123" {
		t.Fatalf("unexpected stdout writer contents: %q", got)
	}
}
