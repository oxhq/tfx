package runfx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/terminal"
	"github.com/oxhq/tfx/writer"
)

type recordingVisual struct {
	stopOnKey bool
	keys      []Key
	resizes   [][2]int
	ticks     int
}

func (v *recordingVisual) Render() []byte { return []byte("record\n") }

func (v *recordingVisual) OnResize(cols, rows int) {
	v.resizes = append(v.resizes, [2]int{cols, rows})
}

func (v *recordingVisual) OnKey(key Key) bool {
	v.keys = append(v.keys, key)
	return v.stopOnKey
}

func (v *recordingVisual) Tick(time.Time) { v.ticks++ }

type escThenErrReader struct {
	sent bool
}

func (r *escThenErrReader) Read(p []byte) (int, error) {
	if !r.sent {
		p[0] = 27
		r.sent = true
		return 1, nil
	}
	return 0, errors.New("boom")
}

func TestKeyReaderEscapeSequencesAndPeekError(t *testing.T) {
	t.Parallel()

	kr := NewKeyReader(bytes.NewBufferString("1;2A"))
	key, err := kr.parseCSISequence()
	if err != nil || key.Code != KeyArrowUp || key.Modifier != ModShift {
		t.Fatalf("unexpected shift-up CSI parse: key=%+v err=%v", key, err)
	}

	kr = NewKeyReader(bytes.NewBufferString("1;2;3A"))
	key, err = kr.parseCSISequence()
	if err != nil || key.Code != KeyUnknown {
		t.Fatalf("expected unknown multi-part CSI, got key=%+v err=%v", key, err)
	}

	kr = NewKeyReader(bytes.NewBufferString(""))
	if _, err := kr.parseCSISequence(); err == nil {
		t.Fatal("expected parseCSISequence to fail on EOF")
	}

	kr = NewKeyReader(bytes.NewBufferString("\x1b"))
	key, err = kr.readKeyBlocking()
	if err != nil || key.Code != KeyEscape {
		t.Fatalf("expected lone escape key, got key=%+v err=%v", key, err)
	}

	kr = NewKeyReader(bytes.NewBufferString("\x1b[A"))
	key, err = kr.readKeyBlocking()
	if err != nil || key.Code != KeyArrowUp {
		t.Fatalf("expected escape sequence to decode arrow up, got key=%+v err=%v", key, err)
	}

	kr = NewKeyReader(bytes.NewBufferString("\x1bx"))
	key, err = kr.readKeyBlocking()
	if err != nil || key.Code != KeyEscape {
		t.Fatalf("expected non-CSI escape to stay as escape, got key=%+v err=%v", key, err)
	}

	kr = NewKeyReader(&escThenErrReader{})
	if _, err := kr.readKeyBlocking(); err == nil ||
		!strings.Contains(err.Error(), "peek escape sequence") {
		t.Fatalf("expected peek error from broken escape reader, got %v", err)
	}
}

func TestKeyCodeStringsAndLoopEventHandling(t *testing.T) {
	t.Parallel()

	names := map[KeyCode]string{
		KeyEnter:      "Enter",
		KeyDelete:     "Delete",
		KeyArrowLeft:  "←",
		KeyArrowRight: "→",
		KeyCtrlC:      "Ctrl+C",
		KeyCtrlD:      "Ctrl+D",
		KeyCtrlZ:      "Ctrl+Z",
		KeyA:          "A",
		KeyZ:          "Z",
		Key0:          "0",
		Key9:          "9",
	}
	for code, want := range names {
		if got := code.String(); got != want {
			t.Fatalf("unexpected key string for %v: got %q want %q", code, got, want)
		}
	}

	loop := &MainLoop{
		writer: writer.NewTerminalWriter(&bytes.Buffer{}, writer.TerminalOptions{}),
		mux:    NewMultiplexer(),
	}
	if _, err := loop.Mount(nil); !errors.Is(err, ErrMountFailed) {
		t.Fatalf("expected ErrMountFailed for nil visual, got %v", err)
	}

	visual := &recordingVisual{}
	unmount, err := loop.Mount(visual)
	if err != nil {
		t.Fatalf("unexpected mount error: %v", err)
	}

	stop, render := loop.handleEvent(keyEvent(Key{Code: KeyEnter}))
	if stop || !render || len(visual.keys) != 1 {
		t.Fatalf(
			"unexpected key event handling: stop=%v render=%v keys=%d",
			stop,
			render,
			len(visual.keys),
		)
	}

	stop, render = loop.handleEvent(tickEvent{time: time.Now()})
	if stop || !render || visual.ticks != 1 {
		t.Fatalf("unexpected tick handling: stop=%v render=%v ticks=%d", stop, render, visual.ticks)
	}

	stop, render = loop.handleEvent(resizeEvent{cols: 80, rows: 24})
	if stop || !render || len(visual.resizes) != 1 {
		t.Fatalf(
			"unexpected resize handling: stop=%v render=%v resizes=%d",
			stop,
			render,
			len(visual.resizes),
		)
	}

	visual.stopOnKey = true
	stop, render = loop.handleEvent(keyEvent(Key{Code: KeyEscape}))
	if !stop || !render {
		t.Fatalf("expected interactive key to stop loop, got stop=%v render=%v", stop, render)
	}

	oldStderr := os.Stderr
	reader, writerPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	defer func() {
		os.Stderr = oldStderr
		_ = reader.Close()
		_ = writerPipe.Close()
	}()
	os.Stderr = writerPipe

	stop, render = loop.handleEvent(errorEvent(errors.New("loop failure")))
	_ = writerPipe.Close()
	data, _ := io.ReadAll(reader)
	if !stop || render || !strings.Contains(string(data), "runfx error: loop failure") {
		t.Fatalf(
			"unexpected error event handling: stop=%v render=%v output=%q",
			stop,
			render,
			string(data),
		)
	}

	stop, render = loop.handleEvent(struct{}{})
	if stop || render {
		t.Fatalf("expected unknown event to be ignored, got stop=%v render=%v", stop, render)
	}

	unmount()
	if got := len(loop.mux.ListVisuals()); got != 0 {
		t.Fatalf("expected unmount closure to remove visual, got %d visuals", got)
	}
}

func TestLoopEventProducers(t *testing.T) {
	t.Parallel()

	keyLoop := &MainLoop{
		reader: NewKeyReader(bytes.NewBuffer(nil)),
		events: make(chan any, 1),
	}
	keyCtx, keyCancel := context.WithCancel(context.Background())
	defer keyCancel()
	go keyLoop.produceKeyEvents(keyCtx)

	select {
	case event := <-keyLoop.events:
		if _, ok := event.(errorEvent); !ok {
			t.Fatalf("expected errorEvent from empty reader, got %T", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected key producer to emit error event")
	}

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	tickLoop := &MainLoop{
		ticker: ticker,
		events: make(chan any, 1),
	}
	tickCtx, tickCancel := context.WithCancel(context.Background())
	defer tickCancel()
	go tickLoop.produceTickEvents(tickCtx)

	select {
	case event := <-tickLoop.events:
		if _, ok := event.(tickEvent); !ok {
			t.Fatalf("expected tickEvent, got %T", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected tick producer to emit event")
	}

	loop := &MainLoop{
		signals: terminal.NewSignalHandler(),
		cancel:  func() {},
	}
	if err := loop.Stop(); err != nil {
		t.Fatalf("unexpected Stop error: %v", err)
	}
}
