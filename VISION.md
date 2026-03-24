# TFX – Vision

TFX was born out of frustration with Go terminal libraries that were either too shallow (`fmt.Println` with colors) or too opinionated (full TUI frameworks like `bubbletea`).  
It aims to fill the space between — **giving CLI authors tools that are elegant, expressive, and composition-friendly** without sacrificing control.

---

## 🎯 The Problem

Most terminal output tools in Go fall into one of three traps:

1. **Too Minimal** – Hardcoded ANSI strings, no structure, no theming.
2. **Too Maximal** – Full UI frameworks that take over the render loop and enforce an app architecture.
3. **Too Fragmented** – One library for color, another for logging, another for progress… no cohesion.

---

## 🌱 The Vision

TFX is a **low-friction, modular terminal toolkit and runtime** designed for
CLI tools, not UIs.

It should feel like:

- `fmt.Println` but prettier.
- `log.Printf` but smarter.
- `chalk`, `ora`, and `slog` had a Go-native baby.

And it should require **zero ceremony** to get started.

At the library level, that means colors, logs, progress, prompts, and render
loops. At the product level, it also means a lightweight wrapper runtime that
can host real tools and workflows without turning into a full TUI framework.

---

## 🧠 Design Beliefs

- **APIs should have layers**: beginner-friendly entrypoints, but power-user depth.
- **Terminal output is storytelling**: progress bars, spinners, and colors aren't fluff — they're UX.
- **No magic, no reflection**: everything should be traceable and debug-friendly.
- **Form follows function, but function should look good.**

---

## 🛠️ Who TFX Is For

- CLI developers who care about polish.
- Go teams who want structure without bloat.
- Builders of tools, utilities, and dev workflows.
- Teams that want to wrap real tools, such as refactoring engines, in a more
  usable terminal runtime.
- Anyone who’s ever used `fmt.Println("...")` and thought: _“this could look better.”_

---

## 🔥 What TFX Will Never Be

- A TUI framework.
- A replacement for the underlying tools it hosts.
- A runtime log server or full observability stack.
- A “just copy this color string” toy lib.
- An everything-and-the-kitchen-sink monolith.

---

## 🧬 Long-term Outlook

TFX is the first in a family of focused tools:

- `tfx` — Terminal Effects
- `cfx` — Config Effects (env, YAML, CLI input)
- `gfx` — Graphics or layout tools (optional)
- `obsfx` — Metrics/log forwarders (optional)

Each one stays lean, single-purpose, composable.

---

> Great tools don’t force choices — they multiply them.
