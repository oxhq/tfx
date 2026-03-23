# TFX v0.1.1

MAINLINE PATCH RELEASE

TFX `v0.1.1` tightens the release story after the inaugural launch: the
project now publishes from `main`, the public docs match the actual package
surface, and local release packaging follows the current version automatically.

## Highlights

- `main` is now the canonical default branch for the project.
- Stale remote release and Codex branches were removed from the public repo.
- The `README` now reflects the real public toolkit shape: `runfx`, `formfx`,
  `flowfx`, `logfx`, `progrefx`, `writer`, `terminal`, and the integrated
  `cmd/tfx` wrapper.
- `make release-artifacts` now resolves release notes from the current
  `VERSION`, so patch releases no longer inherit `v0.1.0` notes by accident.

## What Changed

### Mainline Publishing

- GitHub default branch is now `main`
- Release work now lands and publishes from the canonical branch
- Remote branch noise was removed to keep the public repository clean

### Documentation Accuracy

- Package coverage in `README.md` now includes `runfx`, `formfx`, and `flowfx`
- Feature bullets now include runtime orchestration and the config-driven wrapper
- `cmd/tfx` is described as the integrated wrapper across the full toolkit,
  not just `runfx`

### Release Hygiene

- `VERSION` now reports `v0.1.1`
- Local release bundles copy `docs/release-notes-$(VERSION).md`
- The patch release leaves the `v0.1.0` inaugural tag intact and publishes a
  new patch line instead of rewriting history

## Validation

This release passes:

- `make verify`
- `make release-artifacts`

## Notes

- The packaged wrapper reports `v0.1.1` through `./dist/tfx --version`.
- This is a patch release for correctness and publishing hygiene. The toolkit
  surface introduced in `v0.1.0` remains the same.
