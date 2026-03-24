# `v0.2.0` Release Runbook

This minor release publishes the first broadly dogfooded version of TFX: the
runtime is now exercised outside its own repo, installable across supported
platforms, and packaged with explicit Windows artifacts.

## Version Source

- The canonical release version lives in [`VERSION`](/Users/garaekz/Documents/projects/go/tfx/VERSION).
- `make build-tfx`, `make release-artifacts`, and `make release-archives`
  embed that version into the wrapper binary.
- `go run ./cmd/tfx --version` or `./dist/tfx --version` prints the embedded
  metadata.

## Flows

The root [`tfx.yaml`](/Users/garaekz/Documents/projects/go/tfx/tfx.yaml) exposes:

- `ci`: tests, build, vet
- `quality`: `make verify`
- `release`: `make verify`, `make release-artifacts`, packaged version check,
  and failure diagnostics

## Recommended Release Sequence

1. Confirm [`VERSION`](/Users/garaekz/Documents/projects/go/tfx/VERSION) and
   [`CHANGELOG.md`](/Users/garaekz/Documents/projects/go/tfx/CHANGELOG.md).
2. Run the full local verification path:

```bash
go run ./cmd/tfx --flow quality --run
```

3. Run the wrapper release flow:

```bash
go run ./cmd/tfx --flow release --lane canary --run
```

4. Build the cross-platform archives:

```bash
make release-archives RELEASE_DIR=release
```

5. Inspect the generated outputs:

- `dist/tfx`
- `dist/demo`
- `dist/VERSION`
- `dist/SHA256SUMS`
- `dist/CHANGELOG.md`
- `dist/RELEASE_NOTES.md`
- `release/*.tar.gz`
- `release/*.zip`
- `release/SHA256SUMS`

6. Validate the packaged wrapper explicitly if desired:

```bash
./dist/tfx --version
```

7. Commit, tag, and push:

```bash
git commit -am "release: ship v0.2.0"
git tag -a v0.2.0 -m "v0.2.0"
git push origin main
git push origin v0.2.0
```

## Notes

- The release workflow now uploads both Unix archives and Windows `.zip`
  archives to GitHub releases.
- Interactive runs still require approval for release flows inside the wrapper;
  automation can use `--run` with the configured defaults.
- Secrets remain process-local and are not persisted into the wrapper state
  file.
