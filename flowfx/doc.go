// Package flowfx provides robust, structured, and composable CLI workflow management.
//
// FlowFX is designed to manage deterministic workflows with sequential, parallel,
// branching, wizard-style, scripted, and hierarchical execution models.
//
// # Core Principles
//
//   - Explicit Composition: Users explicitly compose flows; no implicit dependencies
//   - Stateless Execution: Flow state remains contained and explicit
//   - Zero Cross-Imports: No imports from other ecosystem packages except foundational utilities
//   - Predictable and Safe: Structured error handling, retries, timeouts, and cancellation
//   - Non-Interactive Ready: Safe behavior in non-TTY environments
//
// # Basic Usage
//
//	// Sequential execution
//	seq := flowfx.NewSequence().
//		Add(flowfx.NewTask("Setup", setupFunc)).
//		Add(flowfx.NewTask("Process", processFunc)).
//		Add(flowfx.NewTask("Cleanup", cleanupFunc))
//
//	err := seq.Run(ctx)
//
//	// Parallel execution
//	par := flowfx.NewParallel().
//		Add(flowfx.NewTask("Download A", downloadA)).
//		Add(flowfx.NewTask("Download B", downloadB)).
//		Add(flowfx.NewTask("Download C", downloadC))
//
//	err := par.Run(ctx)
//
// # Advanced Features
//
//   - Context-aware cancellation and timeouts
//   - Retry mechanisms with exponential backoff
//   - Progress reporting through injectable interfaces
//   - Conditional branching and wizard-style flows
//   - Hierarchical tree execution
//   - Non-interactive execution support
//
// # Integration
//
// FlowFX integrates with external systems through user-defined interfaces.
// It only imports foundational utilities (terminal, writer, color) and expects
// users to provide integration points for progress reporting, user input, etc.
package flowfx
