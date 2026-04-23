#!/usr/bin/env bash
# shellcheck shell=bash
#
# scripts/lib/paths.sh

set -euo pipefail

source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

# repo-local paths
TMP_DIR="${TMP_DIR:-$ROOT_DIR/tmp}"
BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/build}"

# derive command/binary paths from either:
# - explicit env vars
# - .project.toml
derive_binary_paths() {
  local app=""
  local cmd_base=""

  if [[ -n "${CMD_DIR:-}" ]]; then
    cmd_base="$(basename "$CMD_DIR")"
    [[ "$cmd_base" != "." && "$cmd_base" != "/" && -n "$cmd_base" ]] || {
      die "Refusing: invalid CMD_DIR basename: '$cmd_base'"
    }
  fi

  if [[ -n "${BINARY_NAME:-}" ]]; then
    app="$BINARY_NAME"
  elif [[ -n "$cmd_base" ]]; then
    app="$cmd_base"
  else
    app="$(project_get app)"
  fi

  [[ "$app" != "." && "$app" != "/" && -n "$app" ]] || {
    die "Refusing: invalid binary name: '$app'"
  }

  if [[ -n "$cmd_base" && "$cmd_base" != "$app" ]]; then
    die "Refusing: BINARY_NAME ('$app') does not match CMD_DIR basename ('$cmd_base')"
  fi

  BINARY_NAME="$app"
  CMD_DIR="${CMD_DIR:-$ROOT_DIR/cmd/$BINARY_NAME}"
  BIN_DIR="${BIN_DIR:-$TMP_DIR/bin}"
  BIN_PATH="${BIN_PATH:-$BIN_DIR/$BINARY_NAME}"

  export TMP_DIR BUILD_DIR BINARY_NAME CMD_DIR BIN_DIR BIN_PATH
}

# create repo-local temp directory
ensure_tmp_dir() {
  mkdir -p "$TMP_DIR"
}

# create repo-local build directory
ensure_build_dir() {
  mkdir -p "$BUILD_DIR"
}

# create binary output directory
# caller should run derive_binary_paths first
ensure_bin_dir() {
  [[ -n "${BIN_DIR:-}" ]] || die "Refusing: BIN_DIR is not set. Call derive_binary_paths first."
  mkdir -p "$BIN_DIR"
}
