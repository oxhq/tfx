package runfx

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oxhq/tfx/terminal"
	"github.com/oxhq/tfx/writer"
)

type stopAfterKeyVisual struct {
	renders atomic.Int32
	keys    atomic.Int32
}

func (v *stopAfterKeyVisual) Render() []byte {
	v.renders.Add(1)
	return []byte("frame\n")
}

func (v *stopAfterKeyVisual) OnResize(cols, rows int) {}

func (v *stopAfterKeyVisual) OnKey(key Key) bool {
	v.keys.Add(1)
	return true
}

func TestMainLoopRendersBeforeStoppingOnKey(t *testing.T) {
	pr, pw := io.Pipe()
	defer func() {
		_ = pw.Close()
	}()

	go func() {
		_, _ = pw.Write([]byte("x"))
	}()

	loop := &MainLoop{
		writer:  writer.NewTerminalWriter(&bytes.Buffer{}, writer.TerminalOptions{}),
		reader:  NewKeyReader(pr),
		signals: terminal.NewSignalHandler(),
		mux:     NewMultiplexer(),
		events:  make(chan any, 64),
		ticker:  time.NewTicker(time.Hour),
	}
	defer loop.ticker.Stop()

	visual := &stopAfterKeyVisual{}
	if _, err := loop.Mount(visual); err != nil {
		t.Fatalf("Mount returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- loop.Run(ctx)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run timed out")
	}

	if visual.keys.Load() != 1 {
		t.Fatalf("expected one key event, got %d", visual.keys.Load())
	}

	if visual.renders.Load() < 2 {
		t.Fatalf("expected final render before stop, got %d renders", visual.renders.Load())
	}

	if loop.IsRunning() {
		t.Fatal("loop should not remain running after shutdown")
	}
}
