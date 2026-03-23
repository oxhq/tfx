// Package formfx provides interactive, terminal-based form controls with multipath API support.
//
// FormFX follows the TFX multipath pattern (see MULTIPATH.md):
//   - EXPRESS: Confirm() / Input() / Select()    // Zero-config, quick usage
//   - INSTANTIATED: Confirm(config) / Input(config)  // Config struct for control
//   - DSL: New().Method().Method().Show()         // Chained builder pattern
//
// Interactive features (arrow keys, WASD navigation) are powered by RunFX for robust
// terminal management and graceful fallback in non-TTY environments.
//
// Basic usage:
//
//	// Express: Zero-config
//	result, err := formfx.Confirm("Are you sure?")
//
//	// Instantiated: Config struct
//	cfg := formfx.ConfirmConfig{Label: "Proceed?", Default: true}
//	result, err := formfx.Confirm(cfg)
//
//	// DSL: Chained builder
//	result, err := formfx.NewConfirm().
//		Label("Proceed with deployment?").
//		Default(true).
//		Show()
//
// All prompts return (value, error) and handle context cancellation gracefully.
// In non-TTY environments, prompts fall back to simple text input.
package formfx
