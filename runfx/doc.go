// Package runfx provides a robust, efficient, and highly responsive optional runtime loop
// for advanced terminal (TTY) management, multiplexed visuals, graceful degradation, and detailed observability.
//
// RunFX is designed for explicit mounting of visuals, zero hidden dependencies, and advanced features
// to ensure reliability in complex CLI applications.
//
// # Key Features
//
// - Explicit TTY ownership with advanced multiplexing
// - Cross-platform signal handling (SIGWINCH on Unix, graceful fallback on Windows)
// - Double-buffered, flicker-free rendering
// - Thread-safe visual mounting/unmounting
// - Configurable tick rates for smooth animation (30-120ms)
// - Intelligent fallback for non-TTY environments
// - Zero reflection, global state, or hidden dependencies
// - Multipath API with three entry points for different usage patterns
//
// # Multipath API Usage
//
// RunFX follows TFX's multipath API philosophy with three distinct paths:
//
// ## Beginner Path - Simple & Convenient
//
// Zero-config (uses sensible defaults):
//
//	loop := runfx.Start()
//
// Config struct (explicit configuration):
//
//	cfg := runfx.Config{
//		TickInterval: 30 * time.Millisecond,
//		TestMode:     true,
//	}
//	loop := runfx.Start(cfg)
//
// ## Hardcore Path - Fluent DSL Builder
//
// Declarative chaining for maximum expressiveness:
//
//	loop := runfx.New().
//		SmoothAnimation().        // 30ms tick interval
//		TestMode().
//		Output(os.Stderr).
//		Start()
//
// Available builder methods:
//   - TickInterval(duration) - Custom tick rate
//   - SmoothAnimation()      - 30ms ticks for smooth animations
//   - FastAnimation()        - 100ms ticks for less CPU usage
//   - TestMode()             - Enable test mode for non-TTY environments
//   - Output(writer)         - Custom output destination
//
// ## Experimental Path - Functional Options
//
// Note: This path is experimental and not currently in active use:
//
//	loop := runfx.StartWith(
//		runfx.WithSmoothAnimation(),
//		runfx.WithTestMode(),
//	)
//
// # Basic Usage Pattern
//
// Standard workflow with any of the above creation methods:
//
//	// Create loop (using any method above)
//	loop := runfx.Start()  // or runfx.New().SmoothAnimation().Start()
//
//	// Mount your visual
//	unmount, err := loop.Mount(myVisual)
//	if err != nil {
//		return err
//	}
//	defer unmount()
//
//	// Run with context
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	return loop.Run(ctx)
//
// # Visual Interface
//
// Any type implementing the Visual interface can be mounted:
//
//	type Visual interface {
//		Render(w share.Writer)     // Called during render cycles
//		Tick(now time.Time)        // Called on each tick for animations
//		OnResize(cols, rows int)   // Called when terminal is resized
//	}
//
// # Graceful Degradation
//
// RunFX automatically detects TTY capabilities and falls back to minimal output
// in non-TTY environments, ensuring reliability across different deployment scenarios.
package runfx
