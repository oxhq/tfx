# TFX

> Elegant terminal effects & structured output for Go CLIs — composable, fast, and developer-first.

---

## ✨ What is TFX?

**TFX → TermFX** (short for _Terminal Effects_) is a modular Go toolkit for building expressive, styled, and structured terminal output — without the verbosity and fragmentation of typical Go terminal libraries.

It is _not_ a TUI framework. It is a low-friction, highly composable set of tools for CLIs and utilities that need more than just `fmt.Println`, but less than `bubbletea`.

---

## 🤔 When should you use TFX?

| Tool          | Use Case                               |
| ------------- | -------------------------------------- |
| `fmt.Println` | You don’t care about styling/logging   |
| `log/slog`    | You want structured logs, no styling   |
| `TFX`         | You want structure _and_ style ✨      |
| `bubbletea`   | You’re building a full interactive TUI |

---

## 🧠 TFX Principles

- 💎 _Multi-path API_: same engine, multiple ways to use it.
- 🚫 _No reflection, no runtime surprises_.
- 🧰 _Structured logs with color and context_.
- 🧪 _Built for testing_: capture logs, inject writers.
- 🎨 _Themeable by design_: ANSI + semantic palettes.
- 🧱 _Minimal dependencies_: No third-party bloat.

---

## 🔥 Features

- Unified color system with support for:

  - ANSI
  - 256-color
  - TrueColor (24-bit)
  - Semantic palettes & themes (Dracula, Nord, GitHub, Tailwind)

- Structured logging with:

  - Badge-style tags (\[INFO], \[ERR], etc)
  - Contextual fields
  - Multi-writer support (console + file rotation)
  - JSON, text, and badge formats

- Terminal capability detection:

  - Per-OS fallbacks
  - Unicode/ANSI support
  - CI-awareness, `NO_COLOR`, etc

- Interactive runtime and orchestration with:

  - `runfx` render loops and multiplexed visuals
  - `formfx` prompts, confirms, selects, and validation
  - `flowfx` sequences, branches, retries, scripts, and runners
  - `cmd/tfx` as an integrated wrapper for config-driven pipelines

- Progress bars, spinners, steppers, and tables with smart rendering
- Internal `share/` helpers: `Option[T]`, `Overload[T]` (standardized pattern)

---

## 📦 Packages

| Package           | Description                                               |
| ----------------- | --------------------------------------------------------- |
| `color/`          | Core color system: hex, RGB, ANSI, themes, rendering      |
| `runfx/`          | Lightweight terminal runtime for composable visual loops  |
| `formfx/`         | Prompts, confirms, selects, secrets, and validation       |
| `flowfx/`         | Declarative flow orchestration, scripts, branching, retry |
| `terminal/`       | Terminal detection and capability inference               |
| `logfx/`          | Structured, badge-style logging with writers              |
| `progrefx/`       | Progress bars, spinners, steppers, and tables             |
| `writer/`         | Console & file writers with rotation and theming          |
| `internal/share/` | Internal DX helpers (option sets, overloads, conventions) |

---

## 🚀 Quickstart

```go
import (
    "github.com/oxhq/tfx/color"
    "github.com/oxhq/tfx/logfx"
)

func main() {
    logfx.Success("Server started on port %d", 8080)
    logfx.Badge("DB", "Connected to postgres", color.MaterialGreen)
}
```

Use a spinner:

```go
import (
    "fmt"
    "time"

    "github.com/oxhq/tfx/progrefx"
)

spinner := progrefx.StartSpinner(progrefx.SpinnerConfig{
    Label: "Loading data...",
})

for i := 0; i < 10; i++ {
    spinner.Tick()
    fmt.Print(spinner.Render())
    time.Sleep(100 * time.Millisecond)
}
```

---

## 🖼️ Live Preview

See TFX in action with the demos:

```bash
# Build and run the simple demo
make demo

# Build the versioned wrapper binary
make build-tfx

# Print the wrapper version metadata
go run ./cmd/tfx --version

# Or run the built binary directly
./bin/demo

# Run the integrated wrapper dashboard
go run ./cmd/tfx
```

`cmd/demo` stays as a minimal showcase. `cmd/tfx` is the integrated wrapper
that exercises `runfx`, `formfx`, `flowfx`, `logfx`, and `progrefx`
together.

---

## `cmd/tfx` Config Runner

`cmd/tfx` now loads a project pipeline from `tfx.yaml`, `tfx.yml`,
`.tfx.yaml`, or `.tfx.yml`, searching upward from the current working
directory. A config can define either:

- a single pipeline via top-level `steps`
- multiple named pipelines via `flows` or `profiles`

If no config exists, it falls back to a sensible local pipeline:

- nearest `go.mod`: `go test ./...`, `go build ./...`, and `go vet ./...`
- no Go module: `pwd` and `ls`

Interactive sessions now persist lightweight wrapper state to disk:
the selected flow, selected lane, project name, last run timestamp, and recent
run history are restored between sessions. Secrets stay process-local and are
not written to the state file.

This repository ships its own real root config in
[`tfx.yaml`](/Users/garaekz/Documents/projects/go/tfx/tfx.yaml), so running
`go run ./cmd/tfx` from the repo root gives you actual `ci`, `quality`, and
`release` flows for TFX itself.

Named-flow example:

```yaml
project: tfx
description: TFX local automation
lanes: [preview, canary, production]
working_dir: .
default_flow: ci
flows:
  - name: ci
    description: Fast local verification
    steps:
      - id: test
        title: Run tests
        command: [go, test, ./...]

      - id: build
        title: Build binaries
        command: [go, build, ./...]

  - name: release
    description: Release candidate pipeline
    default_lane: canary
    require_approval: true
    secret_prompt: Release token
    secret_env: TFX_SECRET
    steps:
      - id: test
        title: Run tests
        command: [go, test, ./...]

      - id: vet
        title: Vet packages
        lanes: [canary, production]
        command: [go, vet, ./...]

      - id: coverage
        title: Check coverage threshold
        lanes: [production]
        command: [make, check-coverage]
        continue_on_error: true
```

Behavior notes:

- `working_dir` is resolved relative to the config file directory.
- Top-level and flow-level `before_all`, `after_all`, and `on_failure` hooks are supported.
- Each step `dir` is resolved relative to `working_dir`.
- Each step `command` is executed as argv, not through a shell.
- Each step can declare `artifacts`, and the wrapper also discovers files written under conventional directories such as `dist/`, `coverage/`, `build/`, and `artifacts/`.
- `flows` is the canonical key for multiple named pipelines; `profiles` is a supported alias.
- `default_flow` selects the initial flow; `default_profile` is a supported alias.
- You can preselect a named flow with `go run ./cmd/tfx --flow release`.
- `--profile release` is supported as an alias, and `TFX_FLOW` / `TFX_PROFILE` work too.
- You can preselect a lane with `--lane production` or `TFX_LANE=production`.
- `--run` or `TFX_RUN=true` starts the selected flow immediately without going through the prompts.
- `--json` / `TFX_JSON=true` emits a machine-readable wrapper snapshot with the last run report.
- `--quiet` / `TFX_QUIET=true` emits a compact single-line summary for automation.
- `--version` prints the embedded wrapper version, commit, and build date when available.
- `--flow`, `--lane`, and `--run` compose cleanly, so `tfx --flow release --lane production --run` is valid.
- `TFX_PROJECT` and `TFX_LANE` are exported for every step.
- If `secret_env` is set, the form-collected secret is exported under that name.
- Step `lanes` acts as an allow-list; steps without `lanes` run everywhere.
- `continue_on_error: true` marks the step as failed in the UI but keeps the run going.
- A successful or failed run updates the persisted state and the recent-run list.

For the full schema and a copy-pasteable sample, see
[docs/tfx-yaml.md](./docs/tfx-yaml.md) and [examples/tfx.yaml](./examples/tfx.yaml).
For the concrete release runbook, see [docs/release-v0.1.1.md](./docs/release-v0.1.1.md).
For the release summary itself, see [docs/release-notes-v0.1.1.md](./docs/release-notes-v0.1.1.md).

---

## 🧱 Philosophy

> "If there's only one way to use it, it's not the right way."

TFX promotes a **multi-entry design** for each API:

```go
logfx.Success("Done!")                            // 1. Quick default
logger := logfx.LogWithConfig(logfx.DefaultOptions())
logger.Info("custom")                             // 2. Instantiated style
logfx.If(err).AsWarn().Msg("warn msg")            // 3. DSL / fluent
```

This consistency is achieved via internal helpers like `Overload()` and `Option[T]`.

---

## 📚 Docs

- [VISION.md](./VISION.md) – project intent and market gap
- [DESIGN_GUIDELINES.md](./DESIGN_GUIDELINES.md) – API conventions and patterns
- [ROADMAP.md](./ROADMAP.md) – current status and future plans
- [MULTIPATH.md](./MULTIPATH.md) – why TFX APIs support multiple entry paths
- [docs/tfx-yaml.md](./docs/tfx-yaml.md) – `cmd/tfx` config schema and execution model

---

## 🧪 Status

TFX is under active development. Use at your own risk until `v1.0.0` is tagged.

---

## 📜 License

MIT

---

[![Go Report Card](https://goreportcard.com/badge/github.com/oxhq/tfx)](https://goreportcard.com/report/github.com/oxhq/tfx)

> Built with ☕ and 💢 by [@garaekz](https://github.com/garaekz)
