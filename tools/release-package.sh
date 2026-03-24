#!/usr/bin/env bash

set -euo pipefail

VERSION="${1:?usage: release-package.sh <version> <goos> <goarch> [out_dir] }"
GOOS_TARGET="${2:?usage: release-package.sh <version> <goos> <goarch> [out_dir] }"
GOARCH_TARGET="${3:?usage: release-package.sh <version> <goos> <goarch> [out_dir] }"
OUT_DIR="${4:-release}"

COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
CGO_ENABLED="${CGO_ENABLED:-0}"

archive_base="tfx_${VERSION#v}_${GOOS_TARGET}_${GOARCH_TARGET}"
stage_dir="${OUT_DIR}/${archive_base}"
notes_file="docs/release-notes-${VERSION}.md"

rm -rf "${stage_dir}"
mkdir -p "${stage_dir}"

binary_suffix=""
archive_path="${stage_dir}.tar.gz"
if [[ "${GOOS_TARGET}" == "windows" ]]; then
  binary_suffix=".exe"
  archive_path="${stage_dir}.zip"
fi

ldflags="-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}"

env GOOS="${GOOS_TARGET}" GOARCH="${GOARCH_TARGET}" CGO_ENABLED="${CGO_ENABLED}" \
  go build -trimpath -ldflags "${ldflags}" -o "${stage_dir}/tfx${binary_suffix}" ./cmd/tfx
env GOOS="${GOOS_TARGET}" GOARCH="${GOARCH_TARGET}" CGO_ENABLED="${CGO_ENABLED}" \
  go build -trimpath -ldflags "${ldflags}" -o "${stage_dir}/demo${binary_suffix}" ./cmd/demo

printf "%s\n" "${VERSION}" > "${stage_dir}/VERSION"
cp README.md CHANGELOG.md LICENSE "${stage_dir}/"
if [[ -f "${notes_file}" ]]; then
  cp "${notes_file}" "${stage_dir}/RELEASE_NOTES.md"
fi

if [[ "${GOOS_TARGET}" == "windows" ]]; then
  (
    cd "${OUT_DIR}"
    if command -v zip >/dev/null 2>&1; then
      zip -qr "${archive_base}.zip" "${archive_base}"
    else
      python3 - <<'PY' "${archive_base}"
import pathlib
import sys
import zipfile

archive_base = sys.argv[1]
root = pathlib.Path(".")
target = root / f"{archive_base}.zip"
source = root / archive_base

with zipfile.ZipFile(target, "w", compression=zipfile.ZIP_DEFLATED) as zf:
    for path in source.rglob("*"):
        zf.write(path, path.as_posix())
PY
    fi
  )
else
  tar -C "${OUT_DIR}" -czf "${archive_path}" "${archive_base}"
fi

printf "%s\n" "${archive_path}"
