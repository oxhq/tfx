# TFX v0.1.0

INAUGURAL RELEASE

TFX `v0.1.0` is the first cohesive public release of the toolkit: a modular,
terminal-native Go stack for styled output, structured logs, progress UI,
forms, runtime orchestration, and a self-hosted wrapper that proves the whole
thing can ship real work end to end.

## Highlights

- `cmd/tfx` is now a real integrated wrapper, not a throwaway demo.
- Pipelines run from `tfx.yaml` with named `flows` / `profiles`, `--flow`,
  `--lane`, `--run`, `--json`, `--quiet`, and `--version`.
- Wrapper sessions persist selected flow, lane, project, last run, and recent history.
- Lifecycle hooks (`before_all`, `after_all`, `on_failure`) are first-class.
- Step artifacts can be declared explicitly and are also discovered from common output directories.
- TFX now ships its own repository pipeline and can verify and release itself through TFX.

## What Landed

### Integrated Wrapper

- Live `Overview`, `Flow`, `Forms`, and `Logs` sections in `cmd/tfx`
- Interactive and non-interactive execution paths backed by the same runtime
- Structured machine-readable output for CI and automation

### Modular Toolkit

- `color`, `logfx`, `progrefx`, `runfx`, `flowfx`, `formfx`, `writer`, and `terminal`
- `progrefx.Stepper()` and `progrefx.Table()` as first-class UI primitives
- Deterministic runtime behavior, cleaned public APIs, and a hardened lint/test baseline

### Self-Hosted Delivery

- Root `tfx.yaml` with real `ci`, `quality`, and `release` flows
- Versioned release artifacts produced directly by the wrapper pipeline
- Release bundles with binaries, checksums, changelog, and release notes

## Quick Start

```bash
go run ./cmd/tfx
go run ./cmd/tfx --flow ci --run
go run ./cmd/tfx --flow quality --run --quiet
go run ./cmd/tfx --flow release --lane canary --run
./dist/tfx --version
```

## Validation

This release passes:

- `go test ./...`
- `go build ./...`
- `go vet ./...`
- `golangci-lint run --timeout=5m`
- `go run ./cmd/tfx --flow ci --run --quiet`
- `go run ./cmd/tfx --flow quality --run --quiet`
- `go run ./cmd/tfx --flow release --lane canary --run --quiet`

## Notes

- The packaged wrapper reports `v0.1.0` through `./dist/tfx --version`.
- GitHub releases now have a matching tag workflow that builds multi-platform archives and publishes checksums automatically.
- This is the first release that looks like the original TFX vision: modular, expressive, terminal-native, and self-hosted.
