# Changelog

## Unreleased

Target: `v0.2.0`

### Changed

- Clarified the runtime/product boundary in the public docs, design guidelines,
  and roadmap.
- Promoted `Try*` as the safe contract and added explicit `Must*` wrappers
  across the main multipath APIs.
- Added install docs and scripts plus Windows release packaging support.
- Expanded the dogfooding story from Morfx-only to a generic external-tool
  pattern with real adoption in both `morfx` and `mitl`.

## v0.1.1 - 2026-03-23

### Changed

- Promoted `main` as the canonical default branch and removed stale remote branches.
- Aligned the `README` package map and feature summary with the actual public surface:
  `runfx`, `formfx`, `flowfx`, `progrefx`, and the integrated `cmd/tfx` wrapper.
- Made local release packaging resolve `dist/RELEASE_NOTES.md` from the current
  `VERSION` instead of a hardcoded `v0.1.0` notes file.

### Notes

- This is a patch release focused on release hygiene, naming accuracy, and
  mainline publishing flow.
- No runtime API changes were introduced relative to `v0.1.0`.

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
