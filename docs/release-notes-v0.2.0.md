# TFX v0.2.0

RUNTIME DOGFOODING RELEASE

TFX `v0.2.0` turns the toolkit into a more portable and proven runtime: the
release line now includes install scripts, Windows archives, explicit safe-vs-
strict API entrypoints, and real adoption outside the repo in both `morfx` and
`mitl`.

## Highlights

- `Try*` is now the recoverable contract across the main runtime APIs, with
  explicit `Must*` wrappers for strict call sites.
- Release packaging now covers `linux`, `darwin`, and `windows` on `amd64` and
  `arm64`.
- TFX now documents and demonstrates real dogfooding beyond itself, including
  orchestration of external tools and CI flows in other repos.
- The public docs, roadmap, and design guidance now draw a sharper line between
  “CLI runtime/toolkit” and “application/framework”.

## What Changed

### Safer Runtime Surface

- `runfx`, `progrefx`, `flowfx`, and the shared overload helpers now expose
  clear `Try*` / `Must*` semantics instead of relying on ambiguous short names
  for panic-prone paths.
- The docs now treat the safe path as the default contract and reserve panic
  wrappers for deliberate strict usage.

### Better Distribution

- Release packaging now produces:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/amd64`
  - `darwin/arm64`
  - `windows/amd64`
  - `windows/arm64`
- Install scripts are now documented for Unix-like shells and PowerShell.
- The GitHub release workflow uploads both `.tar.gz` and `.zip` archives, so
  Windows assets are first-class instead of being built and then dropped.

### Real Adoption

- `morfx` documents and uses TFX as a runtime around standalone refactoring
  workflows.
- `mitl` now carries a real `tfx.yaml` and validates core flows through TFX in
  CI.
- TFX’s own docs now describe those adopters directly instead of presenting the
  runtime as only a template or internal demo.

## Validation

This release passes:

- `make verify`
- `go run ./cmd/tfx --flow quality --run --quiet`
- `go run ./cmd/tfx --flow release --lane canary --run --quiet`
- `make release-archives`

## Notes

- The packaged wrapper reports `v0.2.0` through `./dist/tfx --version`.
- The project remains in `v0.x`, but this release is the clearest point yet
  where TFX behaves like a reusable runtime product rather than an internal
  toolkit experiment.
