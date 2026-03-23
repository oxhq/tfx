# `v0.1.1` Release Runbook

This patch release keeps the inaugural `v0.1.0` feature set and publishes the
project cleanly from `main`.

## Version Source

- The canonical release version lives in [`VERSION`](/Users/garaekz/Documents/projects/go/tfx/VERSION).
- `make build-tfx` and `make release-artifacts` embed that version into the wrapper binary.
- `go run ./cmd/tfx --version` or `./dist/tfx --version` prints the embedded metadata.

## Flows

The root [`tfx.yaml`](/Users/garaekz/Documents/projects/go/tfx/tfx.yaml) exposes:

- `ci`: tests, build, vet
- `quality`: `make verify`
- `release`: `make verify`, `make release-artifacts`, packaged version check, failure diagnostics

## Recommended Release Sequence

1. Update [`VERSION`](/Users/garaekz/Documents/projects/go/tfx/VERSION) if needed.
2. Review [`CHANGELOG.md`](/Users/garaekz/Documents/projects/go/tfx/CHANGELOG.md).
3. Run the full local verification path:

```bash
go run ./cmd/tfx --flow quality --run
```

4. Run the release flow:

```bash
go run ./cmd/tfx --flow release --lane canary --run
```

5. Inspect the generated bundle in `dist/`:

- `dist/tfx`
- `dist/demo`
- `dist/VERSION`
- `dist/SHA256SUMS`
- `dist/CHANGELOG.md`
- `dist/RELEASE_NOTES.md`

6. Validate the packaged wrapper explicitly if desired:

```bash
./dist/tfx --version
```

7. Tag and publish once the bundle and notes are acceptable.

## Notes

- `make release-artifacts` now copies release notes based on the current `VERSION`.
- The release flow requires approval inside the wrapper when run interactively.
- In non-interactive automation, `--run` uses the configured defaults and emits either snapshot, `--json`, or `--quiet` output.
- Secrets are still process-local and are not persisted into the wrapper state file.
