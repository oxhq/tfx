# Dogfooding External Tools with `cmd/tfx`

`cmd/tfx` can host more than Morfx. The abstraction is meant to wrap any
repo-local tool or workflow that benefits from structured runs, lane gating,
artifact capture, and a repeatable operator UI.

## Good Fits

This pattern works well for repositories that need to:

- build a standalone CLI or helper binary
- run smoke checks before heavier workflow steps
- execute fixture-driven or snapshot-driven scenarios
- package outputs such as binaries, logs, diffs, or generated files

Examples include:

- code generators
- AST or source transformers
- formatters and linters with repo-local rules
- documentation or asset build pipelines
- release packaging workflows for standalone tools

Real adopters in this workspace now include:

- `morfx`, where TFX wraps standalone verification and release-oriented flows
- `mitl`, where TFX wraps CI verification, `doctor`/preflight checks, and local packaging

## What TFX Should Own

Keep `cmd/tfx` focused on orchestration:

- flow selection
- lane selection
- approvals
- secret injection
- hook execution
- live logs and progress UI
- artifact discovery
- JSON and quiet summaries

Keep the target repository responsible for the tool-specific details:

- actual CLI arguments
- fixture setup
- snapshot assertions
- golden-file updates
- platform-specific build steps

## Recommended Flow Shape

For a standalone external tool repo, a practical setup usually has three flows:

### `ci`

- build the tool
- run the repository test suite
- run a smoke command such as `tool --help` or `tool version`

### `dogfood`

- build the tool
- run fixture or workflow scenarios through a repo-local script
- collect logs, snapshots, or diffs as artifacts

### `release`

- run the strict verification path
- build distributable outputs
- package release artifacts and checksums

## Template Rules

Start from [examples/tfx-external-tool.yaml](../examples/tfx-external-tool.yaml)
and replace the placeholder paths with the target repo's real entrypoints.

The template intentionally keeps the harness generic:

- the binary path is a placeholder
- smoke commands are low-risk
- fixtures are delegated to repo-local scripts
- artifacts are declared explicitly

This keeps the template reusable for Morfx, but also for any other standalone
tool the repo wants to dogfood.
