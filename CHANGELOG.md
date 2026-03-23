# Changelog

## Unreleased

## v0.1.0 - 2026-03-23

### Added

- `cmd/tfx` as an integrated wrapper runner with live `Overview`, `Flow`, `Forms`, and `Logs` sections.
- Config-driven execution through `tfx.yaml` with `flows` / `profiles`, lifecycle hooks, artifacts, `--flow`, `--profile`, `--lane`, and `--run`.
- Disk-backed wrapper state for the selected flow, lane, project, last run, and recent run history.
- Non-interactive snapshot output for automation and CI.
- Machine-readable `--json` output and compact `--quiet` summaries.
- Embedded wrapper version metadata through `VERSION`, `make build-tfx`, and `tfx --version`.
- Runtime visibility for flow lifecycle transitions through the `flowfx`-backed wrapper.
- Artifact-friendly examples that write `coverage.out` and `dist/tfx` from the sample pipelines.
- A real root `tfx.yaml` that runs TFX itself through `ci`, `quality`, and `release` flows.

### Changed

- `--run` now starts the selected flow immediately and works with lane and flow preselection.
- Documentation now reflects the persisted state store, first-class lifecycle hooks, and declared artifacts.

### Notes

- Artifact handling supports both declared step artifacts and convention-based discovery under directories such as `dist/`, `coverage/`, `build/`, and `artifacts/`.
