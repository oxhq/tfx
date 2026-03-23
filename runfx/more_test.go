package runfx

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/terminal"
)

func TestKeyHelpersAndRepresentations(t *testing.T) {
	t.Parallel()

	if !(Key{Code: KeyArrowLeft}).IsArrow() {
		t.Fatal("expected arrow key")
	}
	if !(Key{Code: KeyW}).IsWASD() {
		t.Fatal("expected WASD key")
	}
	if !(Key{Code: Key5}).IsNumber() {
		t.Fatal("expected numeric key")
	}
	if !(Key{Code: KeyTab}).IsNavigation() {
		t.Fatal("expected tab to be navigation")
	}
	if !(Key{Code: KeyEnter}).IsAccept() {
		t.Fatal("expected enter to be accept")
	}
	if !(Key{Code: KeyEscape}).IsSelector() {
		t.Fatal("expected escape to be selector")
	}
	if got := (Key{Code: Key7}).ToNumber(); got != 7 {
		t.Fatalf("unexpected number conversion: %d", got)
	}
	if got := KeyCode(999).String(); got != "Unknown" {
		t.Fatalf("unexpected unknown key string: %q", got)
	}
	if !ModCtrl.Has(ModCtrl) {
		t.Fatal("expected modifier Has() to match")
	}
	if got := (ModCtrl | ModShift).String(); !strings.Contains(got, "Ctrl") ||
		!strings.Contains(got, "Shift") {
		t.Fatalf("unexpected modifier string: %q", got)
	}
}

func TestKeyReaderDecodeCSIAndRegularRune(t *testing.T) {
	t.Parallel()

	kr := NewKeyReader(bytes.NewBuffer(nil))

	key, err := kr.decodeCSI([]byte("A"))
	if err != nil || key.Code != KeyArrowUp {
		t.Fatalf("unexpected simple CSI decode: key=%v err=%v", key.Code, err)
	}
	key, err = kr.decodeCSI([]byte("1;5C"))
	if err != nil || key.Code != KeyArrowRight || key.Modifier != ModCtrl {
		t.Fatalf(
			"unexpected modified CSI decode: key=%v mod=%v err=%v",
			key.Code,
			key.Modifier,
			err,
		)
	}
	key, err = kr.decodeCSI([]byte("3~"))
	if err != nil || key.Code != KeyDelete {
		t.Fatalf("unexpected delete CSI decode: key=%v err=%v", key.Code, err)
	}

	if got := kr.parseRegularRune('A'); got.Code != KeyA || got.Modifier != ModShift {
		t.Fatalf("unexpected uppercase rune parse: %+v", got)
	}
	if got := kr.parseRegularRune('7'); got.Code != Key7 || got.Rune != '7' {
		t.Fatalf("unexpected numeric rune parse: %+v", got)
	}
	if got := kr.parseRegularRune('\t'); got.Code != KeyTab {
		t.Fatalf("unexpected tab parse: %+v", got)
	}
}

func TestRunFXBuildersAndOptions(t *testing.T) {
	t.Parallel()

	input := bytes.NewBufferString("x")
	output := &bytes.Buffer{}

	loop := New().
		TickInterval(time.Second).
		Output(output).
		Input(input).
		SmoothAnimation().
		FastAnimation().
		AutoTick().
		Start()
	if loop == nil {
		t.Fatal("expected builder Start() to create a loop")
	}

	started := Start(Config{TickInterval: time.Second, Output: output, Input: input})
	if started == nil {
		t.Fatal("expected Start() to create a loop")
	}

	cfg := DefaultConfig()
	WithTickInterval(2 * time.Second)(&cfg)
	WithOutput(output)(&cfg)
	WithInput(input)(&cfg)
	WithSmoothAnimation()(&cfg)
	WithFastAnimation()(&cfg)
	WithAutoTick()(&cfg)
	if cfg.Output != output || cfg.Input != input {
		t.Fatalf("expected output/input options to apply, got %+v", cfg)
	}

	started = StartWith(cfg)
	if started == nil {
		t.Fatal("expected StartWith() to create a loop")
	}
}

func TestDetermineTickIntervalAndStop(t *testing.T) {
	t.Parallel()

	if got := determineTickInterval(TTYInfo{IsTTY: false}); got != 250*time.Millisecond {
		t.Fatalf("unexpected non-tty interval: %s", got)
	}
	if got := determineTickInterval(TTYInfo{IsTTY: true, TrueColor: true}); got !=
		16*time.Millisecond {
		t.Fatalf("unexpected truecolor interval: %s", got)
	}
	if got := determineTickInterval(TTYInfo{IsTTY: true, ANSI: true}); got != 33*time.Millisecond {
		t.Fatalf("unexpected ansi interval: %s", got)
	}
	if got := determineTickInterval(TTYInfo{IsTTY: true}); got != 100*time.Millisecond {
		t.Fatalf("unexpected default tty interval: %s", got)
	}

	canceled := false
	loop := &MainLoop{
		signals: terminal.NewSignalHandler(),
		cancel:  func() { canceled = true },
	}
	if err := loop.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !canceled {
		t.Fatal("expected loop cancel function to run")
	}
}

func TestRunFXDebugLogging(t *testing.T) {
	t.Parallel()

	debugMode = false
	defer func() { debugMode = false }()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v", err)
	}
	defer func() {
		os.Stderr = oldStderr
		_ = r.Close()
		_ = w.Close()
	}()
	os.Stderr = w

	EnableDebug()
	DebugLog("hello %s", "world")
	LogMount("visual")
	LogUnmount("visual")
	LogRender("visual")
	LogTick(1, 2, time.Millisecond)
	LogFlush(1, 2*time.Millisecond)
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	output := string(data)
	if !strings.Contains(output, "[RunFX DEBUG] hello world") ||
		!strings.Contains(output, "Mounted visual") {
		t.Fatalf("unexpected debug output: %q", output)
	}
}

func TestKeyReaderReadKeyContextCancel(t *testing.T) {
	t.Parallel()

	reader := NewKeyReader(bytes.NewBuffer(nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := reader.ReadKey(ctx); err == nil {
		t.Fatal("expected canceled context error")
	}
}
