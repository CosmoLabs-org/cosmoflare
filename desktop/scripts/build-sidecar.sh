#!/usr/bin/env bash
#
# build-sidecar.sh — cross-compile the cosmoflare Go daemon as a Tauri
# `externalBin` sidecar, one binary per target triple. Tauri resolves the
# sidecar by appending the host triple to the basename, so each output is
# `cosmoflare-<target-triple>` (plus `.exe` on Windows).
#
# Pure-Go (CGO_ENABLED=0) → fully static, cross-compiles to every target from
# any host with no C toolchain. Stripped (-s -w) for smaller bundles.
#
# Usage:
#   build-sidecar.sh            # all four triples (release / CI)
#   build-sidecar.sh --host     # just the host triple (fast `tauri dev`)
#
# Wired as Tauri's beforeBuildCommand (see tauri.conf.json) so the sidecar is
# present before bundling.

set -euo pipefail

# Repo root (this script lives at desktop/scripts/build-sidecar.sh).
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/desktop/src-tauri/binaries"
mkdir -p "$OUT"

LDFLAGS="-s -w"

# triple|goos|goarch
TARGETS=(
  "aarch64-apple-darwin|darwin|arm64"
  "x86_64-apple-darwin|darwin|amd64"
  "x86_64-pc-windows-msvc|windows|amd64"
  "x86_64-unknown-linux-gnu|linux|amd64"
)

HOST_ONLY=0
if [ "${1:-}" = "--host" ]; then
  HOST_ONLY=1
fi

build_one() {
  local triple="$1" goos="$2" goarch="$3"
  local out="$OUT/cosmoflare-$triple"
  if [ "$goos" = "windows" ]; then
    out="$out.exe"
  fi
  printf '  -> %-28s ' "$triple"
  GOWORK=off CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go -C "$ROOT" build -trimpath -ldflags "$LDFLAGS" -o "$out" .
  echo "ok ($(du -h "$out" | cut -f1))"
}

echo "Building cosmoflare sidecars into desktop/src-tauri/binaries/"

if [ "$HOST_ONLY" = "1" ]; then
  case "$(uname -s)-$(uname -m)" in
    Darwin-arm64)  build_one aarch64-apple-darwin     darwin arm64 ;;
    Darwin-x86_64) build_one x86_64-apple-darwin      darwin amd64 ;;
    Linux-x86_64)  build_one x86_64-unknown-linux-gnu linux  amd64 ;;
    MINGW*-x86_64|*-x86_64) build_one x86_64-pc-windows-msvc windows amd64 ;;
    *) echo "unknown host triple; building all targets" ; HOST_ONLY=0 ;;
  esac
fi

if [ "$HOST_ONLY" = "0" ]; then
  for spec in "${TARGETS[@]}"; do
    IFS='|' read -r triple goos goarch <<<"$spec"
    build_one "$triple" "$goos" "$goarch"
  done
fi

echo "Sidecar build complete."
