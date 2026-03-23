# TFX – Roadmap

This document outlines the current feature set, future plans, and priorities for TFX.

> 🧠 **Note:** TFX is maintained by a full-time engineer ([@garaekz](https://github.com/garaekz)) in personal time.  
> Development is driven by intent, quality, and bursts of ADHD-fueled productivity.  
> **Some features may land in 48 hours. Others might wait two months.**  
> Stability and vision come first — not hype cycles.

---

## ✅ Implemented

### 🎨 Color System

- ANSI, 256-color, and TrueColor support
- Semantic themes: Dracula, Nord, GitHub, Tailwind, Material
- Rainbow, gradient, and glow effects
- Palette composition and utilities

### 🖥️ Terminal Detection

- Smart fallback system for terminals with limited support
- `NO_COLOR` and CI/CD detection
- Unicode-safe checks and symbol substitutions
- Cross-platform (Linux/macOS/Windows)

### 🧱 Core Logging

- `logfx` package with multi-writer structured logs
- Badge-style logging (e.g. `[INFO]`, `[ERR]`, etc)
- Color-aware writers: console + file with rotation
- Contextual logging via `.WithFields()` and `.WithRequestID()`
- Format options: plain, JSON, badge
- Level filtering and output hooks

### 🌀 Writers System

- Console writer
- File writer with rotation
- MultiWriter / Async / Filtered writer
- Shared writer factory with flush control

### ⏳ Progress & Spinners

- Themed spinners with success/error states
- Progress bars with label, width, and style options
- Support for rainbow/gradient/solid fill styles
- Multi-step progress flow via `progrefx.Stepper()`
- Grid-style terminal tables via `progrefx.Table()`

### 🧩 Wrapper & Demos

- `cmd/demo` minimal showcase binary
- `cmd/tfx` integrated wrapper/dashboard using `runfx`
- Config-driven `cmd/tfx` runner via `tfx.yaml` with fallback local pipelines
- Named `cmd/tfx` flows/profiles selectable from the form workflow
- Disk-backed wrapper state for flow, lane, project, and recent run history
- Non-interactive `--run` mode with `--flow`/`--lane` selection for direct automation
- Machine-readable `--json`, compact `--quiet`, and embedded `--version` output
- First-class lifecycle hooks and declared/discovered artifact reporting
- Real root `tfx.yaml` that runs the TFX repo itself through `ci`, `quality`, and `release` flows
- Flow lifecycle visibility through the live wrapper and non-TTY snapshots

### 🧪 Testing Infrastructure

- Test capture writer for log assertion
- Progress/spinner test mode
- `Makefile` + `tools/test.sh` workflow

### 📚 Documentation & Philosophy

- `README.md` with live preview and usage examples
- `VISION.md`, `ROADMAP.md`, `THEMES.md`, `MULTIPATH.md`
- Design guidelines: DX-first, no reflection, multi-path APIs

---

## Immediate Priorities before Launch

- ✅ Achieve 100% test coverage on all core packages
- 🧪 Add edge case tests to ensure robustness and correctness
- 🛠️ Stabilize and optimize existing features for real-world usage
- 📖 Expand documentation with clear examples and usage patterns
- 🔍 Perform full code review pass for maintainability and DX
- 🎯 Finalize public APIs and remove experimental/unpolished code
- 🏗️ Enforce race-safe builds/tests via `Makefile` and CI
- 🗂️ Refactor project structure for clarity and navigation
- 📦 Publish a polished release note and changelog for the integrated wrapper
- 📬 Incorporate early user feedback to guide roadmap
- 🚀 Deliver a polished, minimal, opinionated `v0.1.0`

## 🧠 Planned (Post-v0.1.x)

- `logfx.Trace()` level (hidden by default)
- Theme preview playground (`themes_preview.go`)
- Spinners with alternate glyph sets (e.g. braille, dot, arrows)
- Theme-based progress/spinner layout presets
- Richer artifact publishing targets and export helpers for wrapper runs
- Optional remote or team-shared wrapper state backends
- Dynamic runtime theme switching
- ANSI art templates / block layouts
- Lightweight metrics (counters, timers)

---

## 🧪 Optional / Experimental

- TFX-compatible plugins or "addons"
- Themed terminal banners
- Emoji/picto substitution fallback
- Benchmarks and internal metrics
- Color contrast validation helpers

---

## ⛔ Out of Scope

> These features are intentionally left out of the core toolkit:

- ❌ General-purpose configuration systems (handled by `configfx`, `cfx`)
- ❌ Full TUI rendering / state management
- ❌ Reflection-based option handling
- ❌ Remote telemetry, hosted services, or vendor lock-in

---

> TFX is for CLI authors who want power without ceremony,  
> and polish without bloat.
