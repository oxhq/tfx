package runfx

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/oxhq/tfx/terminal"
	"github.com/oxhq/tfx/writer"
	"golang.org/x/term"
)

// --- Event Types ---
type (
	keyEvent    Key
	tickEvent   struct{ time time.Time }
	resizeEvent struct{ cols, rows int }
	errorEvent  error
)

// --- Loop Definition ---

// MainLoop is handles terminal I/O, signals, and the render/tick cycle.
type MainLoop struct {
	// Configuration and Dependencies
	writer  *writer.TerminalWriter
	reader  *KeyReader
	signals *terminal.SignalHandler
	mux     *Multiplexer
	ticker  *time.Ticker
	// tickInterval is stored until Run starts so constructing a loop has no side effects.
	tickInterval time.Duration

	// Internal State
	events   chan any // Central event channel
	cancel   context.CancelFunc
	rawState *term.State
	running  atomic.Bool
}

// --- Public API Methods ---

// Mount registers a visual component with the loop's multiplexer.
func (ml *MainLoop) Mount(v Visual) (unmount func(), err error) {
	if v == nil {
		return nil, ErrMountFailed
	}

	// Get the current terminal size and inform the new visual immediately.
	// This ensures the component has its layout calculated before the first render.
	if cols, rows, err := ml.writer.GetSize(); err == nil {
		v.OnResize(cols, rows)
	}

	// Mount the visual in the multiplexer and get its unique ID.
	id := ml.mux.Mount(v)

	// Return a closure that captures the ID to unmount the visual later.
	return func() { ml.mux.Unmount(id) }, nil
}

// Run starts the main loop and blocks until the context is canceled or Stop() is called.
func (ml *MainLoop) Run(ctx context.Context) error {
	if !ml.running.CompareAndSwap(false, true) {
		return ErrLoopAlreadyRunning
	}
	defer ml.running.Store(false)

	// Setup terminal
	if state, err := ml.writer.EnableRawMode(); err == nil {
		ml.rawState = state
		defer func() {
			_ = ml.writer.RestoreMode(ml.rawState)
		}()
	}
	_ = ml.writer.HideCursor()
	defer func() {
		_ = ml.writer.ShowCursor()
	}()
	defer func() {
		_ = ml.writer.Clear()
	}()

	// Create a cancellable context for the loop's goroutines
	loopCtx, cancel := context.WithCancel(ctx)
	ml.cancel = cancel
	defer func() {
		ml.signals.Stop()
		cancel()
	}()
	createdTicker := false
	if ml.ticker == nil {
		tickInterval := ml.tickInterval
		if tickInterval <= 0 {
			tickInterval = DefaultConfig().TickInterval
		}
		ml.ticker = time.NewTicker(tickInterval)
		createdTicker = true
	}
	defer func() {
		if createdTicker {
			ml.ticker.Stop()
			ml.ticker = nil
		}
	}()

	// Start event producers
	go ml.produceKeyEvents(loopCtx)
	go ml.produceTickEvents(loopCtx)
	go ml.produceSignalEvents(loopCtx)

	// Initial render
	ml.renderFrame()

	// Main event processing loop
	for {
		select {
		case <-loopCtx.Done():
			return loopCtx.Err()
		case e := <-ml.events:
			shouldStop, shouldRender := ml.handleEvent(e)
			if shouldRender {
				ml.renderFrame()
			}
			if shouldStop {
				return nil
			}
		}
	}
}

// Stop gracefully shuts down the main loop.
func (ml *MainLoop) Stop() error {
	if ml.signals != nil {
		ml.signals.Stop()
	}
	if ml.cancel != nil {
		ml.cancel()
	}
	return nil
}

// IsRunning checks if the loop is currently active.
func (ml *MainLoop) IsRunning() bool {
	return ml.running.Load()
}

// --- Internal Event Producers ---

func (ml *MainLoop) produceKeyEvents(ctx context.Context) {
	for {
		key, err := ml.reader.ReadKey(ctx)
		if err != nil {
			select {
			case ml.events <- errorEvent(err):
			case <-ctx.Done():
				return
			}
			return
		}
		select {
		case ml.events <- keyEvent(key):
		case <-ctx.Done():
			return
		}
	}
}

func (ml *MainLoop) produceTickEvents(ctx context.Context) {
	for {
		select {
		case t := <-ml.ticker.C:
			select {
			case ml.events <- tickEvent{time: t}:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (ml *MainLoop) produceSignalEvents(ctx context.Context) {
	ml.signals.OnResize(func() {
		cols, rows, err := ml.writer.GetSize()
		if err == nil {
			select {
			case ml.events <- resizeEvent{cols: cols, rows: rows}:
			case <-ctx.Done():
			}
		}
	})
	ml.signals.OnStop(func() {
		_ = ml.Stop()
	})
	ml.signals.Listen(ctx)
}

// --- Internal Event Handler and Renderer ---

// handleEvent processes a single event from the central channel.
// It returns (shouldStop, shouldRender).
func (ml *MainLoop) handleEvent(e any) (bool, bool) {
	switch event := e.(type) {
	case keyEvent:
		// Dispatch key to all interactive visuals using their IDs.
		for _, id := range ml.mux.ListVisuals() {
			if v, ok := ml.mux.GetVisual(id); ok {
				if i, isInteractive := v.(Interactive); isInteractive {
					if i.OnKey(Key(event)) {
						// Stop if OnKey returns true. Render one last time.
						return true, true
					}
				}
			}
		}
		// If no component stopped the loop, we assume a state change and re-render.
		return false, true
	case tickEvent:
		// Dispatch tick to all updatable visuals.
		for _, id := range ml.mux.ListVisuals() {
			if v, ok := ml.mux.GetVisual(id); ok {
				if u, isUpdatable := v.(Updatable); isUpdatable {
					u.Tick(event.time)
				}
			}
		}
		// A tick always implies a potential visual change.
		return false, true
	case resizeEvent:
		// Dispatch resize to all visuals.
		ml.mux.OnResize(event.cols, event.rows)
		// A resize always requires a full re-render.
		return false, true
	case errorEvent:
		// Log or handle error, for now we stop.
		fmt.Fprintf(os.Stderr, "runfx error: %v\n", event)
		return true, false // Stop, no need to render.
	}
	// Default case for unknown events.
	return false, false
}

// renderFrame clears the screen and renders all mounted visuals.
func (ml *MainLoop) renderFrame() {
	_ = ml.writer.Clear()
	// The multiplexer now handles aggregating the render output
	renderBytes := ml.mux.Render()
	_, _ = ml.writer.Write(renderBytes)
	_ = ml.writer.Flush()
}
