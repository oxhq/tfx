package runfx

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/oxhq/tfx/terminal"
	"github.com/oxhq/tfx/writer"
)

func TestRunFXAdditionalKeyAndBuilderCoverage(t *testing.T) {
	t.Parallel()

	if got := (Key{Code: KeyUnknown}).ToNumber(); got != -1 {
		t.Fatalf("expected non-number conversion to return -1, got %d", got)
	}
	if got := ModNone.String(); got != "None" {
		t.Fatalf("unexpected ModNone string: %q", got)
	}

	names := map[KeyCode]string{
		KeyEscape:    "Escape",
		KeyBackspace: "Backspace",
		KeyTab:       "Tab",
		KeySpace:     "Space",
		KeyArrowUp:   "↑",
		KeyArrowDown: "↓",
	}
	for code, want := range names {
		if got := code.String(); got != want {
			t.Fatalf("unexpected key string for %v: got %q want %q", code, got, want)
		}
	}

	reader := NewKeyReader(nil)
	if reader.input != os.Stdin {
		t.Fatal("expected nil input to default to os.Stdin")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected Start to panic on invalid multipath input")
		}
	}()
	_ = Start("bad")
}

func TestMustStartPanicsOnInvalidMultipathInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected MustStart to panic on invalid multipath input")
		}
	}()

	_ = MustStart("bad")
}

func TestRunFXDecodeCSIAndRunesMoreBranches(t *testing.T) {
	t.Parallel()

	kr := NewKeyReader(bytes.NewBuffer(nil))

	tests := []struct {
		seq      []byte
		wantCode KeyCode
		wantMod  Modifier
	}{
		{seq: []byte("1;3B"), wantCode: KeyArrowDown, wantMod: ModAlt},
		{seq: []byte("1;4D"), wantCode: KeyArrowLeft, wantMod: ModShift | ModAlt},
		{seq: []byte("1;6A"), wantCode: KeyArrowUp, wantMod: ModCtrl | ModShift},
		{seq: []byte("1;7C"), wantCode: KeyArrowRight, wantMod: ModCtrl | ModAlt},
		{seq: []byte("1;8D"), wantCode: KeyArrowLeft, wantMod: ModCtrl | ModAlt | ModShift},
	}
	for _, tt := range tests {
		key, err := kr.decodeCSI(tt.seq)
		if err != nil || key.Code != tt.wantCode || key.Modifier != tt.wantMod {
			t.Fatalf("unexpected CSI decode for %q: key=%+v err=%v", string(tt.seq), key, err)
		}
	}

	if key, err := kr.decodeCSI([]byte("1;9D")); err != nil || key.Code != KeyArrowLeft ||
		key.Modifier != ModNone {
		t.Fatalf(
			"expected unknown modifier number to fall back to no modifier, got key=%+v err=%v",
			key,
			err,
		)
	}
	if key, err := kr.decodeCSI([]byte("1;2~")); err != nil || key.Code != KeyUnknown {
		t.Fatalf("expected unknown CSI suffix to return KeyUnknown, got key=%+v err=%v", key, err)
	}

	runeTests := []struct {
		r        rune
		wantCode KeyCode
		wantRune rune
		wantMod  Modifier
	}{
		{r: '\n', wantCode: KeyEnter},
		{r: ' ', wantCode: KeySpace, wantRune: ' '},
		{r: 8, wantCode: KeyBackspace},
		{r: 3, wantCode: KeyCtrlC, wantMod: ModCtrl},
		{r: 4, wantCode: KeyCtrlD, wantMod: ModCtrl},
		{r: 26, wantCode: KeyCtrlZ, wantMod: ModCtrl},
		{r: 'a', wantCode: KeyA, wantRune: 'a'},
		{r: '?', wantCode: KeyUnknown, wantRune: '?'},
	}
	for _, tt := range runeTests {
		got := kr.parseRegularRune(tt.r)
		if got.Code != tt.wantCode || got.Rune != tt.wantRune || got.Modifier != tt.wantMod {
			t.Fatalf("unexpected parsed rune for %q: %+v", string(tt.r), got)
		}
	}
}

func TestRunFXLoopDefaultsAndGuards(t *testing.T) {
	t.Parallel()

	loop, ok := newLoopWithConfig(Config{}).(*MainLoop)
	if !ok {
		t.Fatal("expected newLoopWithConfig to return *MainLoop")
	}
	if loop.writer == nil || loop.reader == nil || loop.signals == nil || loop.mux == nil ||
		loop.events == nil {
		t.Fatalf("expected loop dependencies to be initialized, got %+v", loop)
	}
	if loop.reader.input != os.Stdin {
		t.Fatal("expected default loop reader input to be os.Stdin")
	}

	runningLoop := &MainLoop{}
	runningLoop.running.Store(true)
	if err := runningLoop.Run(context.Background()); !errors.Is(err, ErrLoopAlreadyRunning) {
		t.Fatalf("expected already-running error, got %v", err)
	}

	cancelable := &MainLoop{
		writer:  writer.NewTerminalWriter(&bytes.Buffer{}, writer.TerminalOptions{}),
		reader:  NewKeyReader(bytes.NewBuffer(nil)),
		signals: terminal.NewSignalHandler(),
		mux:     NewMultiplexer(),
		events:  make(chan any, 8),
		ticker:  time.NewTicker(time.Hour),
	}
	defer cancelable.ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cancelable.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context from Run, got %v", err)
	}
}

func TestProduceSignalEventsHandlesResizeSignal(t *testing.T) {
	loop := &MainLoop{
		writer:  writer.NewTerminalWriter(&bytes.Buffer{}, writer.TerminalOptions{}),
		signals: terminal.NewSignalHandler(),
		events:  make(chan any, 8),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		loop.produceSignalEvents(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatalf("failed to send SIGWINCH: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("signal producer did not stop after context cancellation")
	}

	select {
	case event := <-loop.events:
		if _, ok := event.(resizeEvent); !ok {
			t.Fatalf("expected resizeEvent after SIGWINCH, got %T", event)
		}
	default:
		// Non-interactive environments may not be able to resolve a terminal size here.
	}
}

func TestReadKeyPropagatesReaderErrors(t *testing.T) {
	t.Parallel()

	reader := NewKeyReader(bytes.NewBuffer(nil))
	ctx := context.Background()
	key, err := reader.ReadKey(ctx)
	if err == nil || !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("expected EOF from empty reader, got key=%+v err=%v", key, err)
	}
}
