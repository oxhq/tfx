#!/usr/bin/env bash

set -euo pipefail

REPO="${TFX_REPO:-oxhq/tfx}"
INSTALL_DIR="${TFX_INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="${TFX_VERSION:-}"
TMP_DIR=""

usage() {
  cat <<'EOF'
Usage: install.sh [-v version] [-d install_dir] [-r repo]

Installs the TFX wrapper binary from GitHub releases.

Examples:
  ./tools/install.sh
  ./tools/install.sh -v v0.1.1
  ./tools/install.sh -d /usr/local/bin
EOF
}

cleanup() {
  if [[ -n "${TMP_DIR}" && -d "${TMP_DIR}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT

while getopts ":v:d:r:h" opt; do
  case "${opt}" in
    v) VERSION="${OPTARG}" ;;
    d) INSTALL_DIR="${OPTARG}" ;;
    r) REPO="${OPTARG}" ;;
    h)
      usage
      exit 0
      ;;
    :)
      echo "missing value for -${OPTARG}" >&2
      usage >&2
      exit 1
      ;;
    \?)
      echo "unknown option: -${OPTARG}" >&2
      usage >&2
      exit 1
      ;;
  esac
done

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd tar

detect_os() {
  case "$(uname -s)" in
    Linux) printf "linux" ;;
    Darwin) printf "darwin" ;;
    *)
      echo "unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf "amd64" ;;
    arm64|aarch64) printf "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

resolve_latest_version() {
  local url effective
  url="https://github.com/${REPO}/releases/latest"
  effective="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "${url}")"
  basename "${effective}"
}

if [[ -z "${VERSION}" ]]; then
  VERSION="$(resolve_latest_version)"
fi

OS_NAME="$(detect_os)"
ARCH_NAME="$(detect_arch)"
ASSET="tfx_${VERSION#v}_${OS_NAME}_${ARCH_NAME}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

TMP_DIR="$(mktemp -d)"
ARCHIVE_PATH="${TMP_DIR}/${ASSET}"

echo "Downloading ${URL}"
curl -fsSL "${URL}" -o "${ARCHIVE_PATH}"
tar -xzf "${ARCHIVE_PATH}" -C "${TMP_DIR}"

mkdir -p "${INSTALL_DIR}"
cp "${TMP_DIR}/tfx_${VERSION#v}_${OS_NAME}_${ARCH_NAME}/tfx" "${INSTALL_DIR}/tfx"
chmod +x "${INSTALL_DIR}/tfx"

echo "Installed tfx ${VERSION} to ${INSTALL_DIR}/tfx"
