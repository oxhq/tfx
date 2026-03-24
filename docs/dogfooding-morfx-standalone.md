# Dogfooding `morfx` as a Standalone Tool with `cmd/tfx`

If the dogfooding target is `morfx`, use the standalone binary path, not the
MCP adapter.

That is the better test for `cmd/tfx` because it validates the wrapper against
the same execution model that users will run locally or in CI:

- build a real CLI binary
- run real commands against fixtures
- collect logs, snapshots, or diffs
- package the output as artifacts

## Recommended Shape

For a standalone `morfx` repository, the wrapper should normally expose at
least these flows:

### `ci`

- build the CLI
- run repository tests
- run a smoke command such as `morfx --help`

### `dogfood`

- build the CLI
- run fixture scenarios through a repo-local script or command harness
- collect generated diffs, snapshots, or result logs as artifacts

### `release`

- run the stricter verification path
- build distributable binaries
- package checksums, notes, and release outputs

## Why a Harness is Better than Inline CLI Guesses

For AST-editing tools, the exact CLI flags tend to move faster than the wrapper
configuration. A repo-local harness keeps that knowledge close to the tool.

Instead of hardcoding every edit command into `tfx.yaml`, prefer steps like:

- `./bin/morfx --help`
- `./scripts/dogfood-fixtures.sh`
- `./scripts/assert-snapshots.sh`

That keeps `tfx.yaml` stable while the standalone tool evolves.

## What TFX Should Own

Let `cmd/tfx` own:

- flow selection
- lane selection
- approvals
- secret injection
- hook execution
- artifact discovery
- JSON and quiet summaries
- live logs and stepper/table progress

Let the `morfx` repository own:

- actual CLI arguments
- fixture setup
- snapshot assertions
- golden-file updates
- language-specific test matrices

## Template

Start from [examples/tfx-morfx-standalone.yaml](../examples/tfx-morfx-standalone.yaml)
if Morfx is the target, or [examples/tfx-external-tool.yaml](../examples/tfx-external-tool.yaml)
for a generic external-tool template.

The template intentionally uses:

- a real binary build step
- a smoke step
- a repo-local fixture harness step
- artifact declarations for logs and packaged binaries

This is the right shape for dogfooding `morfx` as a standalone product rather
than as an MCP backend.
