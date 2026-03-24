package runfx

import (
	"io"
	"os"
	"time"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/terminal"
	"github.com/oxhq/tfx/writer"
)

// --- MULTIPATH API FUNCTIONS ---

// MustStart creates a new Loop with multipath configuration support and panics
// on invalid multipath input.
//
// Prefer TryStart when configuration may be user-provided or otherwise fallible.
func MustStart(opts ...any) Loop {
	loop, err := TryStart(opts...)
	if err != nil {
		panic(err)
	}
	return loop
}

// Start creates and starts a new Loop with multipath configuration support.
//
// Start is kept as a zero-ceremony compatibility alias for MustStart. Prefer
// TryStart for fallible input or MustStart when panic semantics should be
// explicit at the callsite.
func Start(opts ...any) Loop {
	return MustStart(opts...)
}

// TryStart creates a new Loop with multipath configuration support without panicking.
func TryStart(opts ...any) (Loop, error) {
	cfg, err := share.TryOverloadWithOptions(opts, DefaultConfig())
	if err != nil {
		return nil, err
	}
	return newLoopWithConfig(cfg), nil
}

// newLoopWithConfig creates a new Loop with the given configuration
func newLoopWithConfig(cfg Config) Loop {
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}
	input := cfg.Input
	if input == nil {
		input = os.Stdin
	}

	ttyFile := resolveTTYFile(output, input)
	ttyInfo := DetectTTYForIO(output, input)
	tw := writer.NewTerminalWriter(output, writer.TerminalOptions{
		DoubleBuffer: true,
		DisableColor: ttyInfo.NoColor,
		TTYFile:      ttyFile,
	})

	return &MainLoop{
		mux:          NewMultiplexer(),
		writer:       tw,
		reader:       NewKeyReader(input),
		signals:      terminal.NewSignalHandler(),
		events:       make(chan any, 64), // buffered channel for events
		tickInterval: cfg.TickInterval,
	}
}

// --- DSL BUILDER API (Hardcore Path) ---

// LoopBuilder provides a fluent DSL interface for configuring RunFX loops
type LoopBuilder struct {
	config Config
}

// New creates a new LoopBuilder with default configuration.
// HARDCORE path - provides fluent DSL interface:
//
//	runfx.New().TickInterval(100*time.Millisecond).TestMode(true).Start()
func New() *LoopBuilder {
	return &LoopBuilder{config: DefaultConfig()}
}

// TickInterval sets the tick interval for the event loop
func (b *LoopBuilder) TickInterval(interval time.Duration) *LoopBuilder {
	b.config.TickInterval = interval
	return b
}

// Output sets the output writer
func (b *LoopBuilder) Output(output io.Writer) *LoopBuilder {
	b.config.Output = output
	return b
}

// Input sets the keyboard input source.
func (b *LoopBuilder) Input(input io.Reader) *LoopBuilder {
	b.config.Input = input
	return b
}

// SmoothAnimation sets tick interval to 30ms for very smooth animations
func (b *LoopBuilder) SmoothAnimation() *LoopBuilder {
	b.config.TickInterval = 30 * time.Millisecond
	return b
}

// FastAnimation sets tick interval to 100ms for faster/less resource intensive animations
func (b *LoopBuilder) FastAnimation() *LoopBuilder {
	b.config.TickInterval = 100 * time.Millisecond
	return b
}

// Start creates and returns the configured Loop instance
func (b *LoopBuilder) Start() Loop {
	return newLoopWithConfig(b.config)
}

func (b *LoopBuilder) AutoTick() *LoopBuilder {
	tty := DetectTTY()
	b.config.TickInterval = determineTickInterval(tty)
	return b
}

// --- Vía 4 (Functional Options) ---
// This path provides a composable way to configure a loop.
// As noted in the TFX philosophy, this is often considered an alternative to the DSL path.

// StartWith creates and returns a new Loop configured with the provided functional options.
// It is the primary entry point for the Functional Options path.
func StartWith(cfg Config) Loop {
	return newLoopWithConfig(cfg)
}

// WithTickInterval returns an Option to set a custom tick interval.
func WithTickInterval(interval time.Duration) share.Option[Config] {
	return func(cfg *Config) {
		cfg.TickInterval = interval
	}
}

// WithOutput returns an Option to set a custom output writer.
func WithOutput(output io.Writer) share.Option[Config] {
	return func(cfg *Config) {
		cfg.Output = output
	}
}

// WithInput returns an Option to set a custom input source.
func WithInput(input io.Reader) share.Option[Config] {
	return func(cfg *Config) {
		cfg.Input = input
	}
}

// WithSmoothAnimation returns an Option to set a 30ms tick interval for smooth animations.
func WithSmoothAnimation() share.Option[Config] {
	return func(cfg *Config) {
		cfg.TickInterval = 30 * time.Millisecond
	}
}

// WithFastAnimation returns an Option to set a 100ms tick interval for efficient animations.
func WithFastAnimation() share.Option[Config] {
	return func(cfg *Config) {
		cfg.TickInterval = 100 * time.Millisecond
	}
}

// WithAutoTick returns an Option to set the tick interval based on detected TTY capabilities.
func WithAutoTick() share.Option[Config] {
	return func(cfg *Config) {
		// Note: This requires access to the output writer from the config.
		// If the output is changed by another option, the order matters.
		tty := DetectTTYForOutput(cfg.Output)
		cfg.TickInterval = determineTickInterval(tty)
	}
}
