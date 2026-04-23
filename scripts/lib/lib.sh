#!/usr/bin/env bash
# shellcheck shell=bash
#
# scripts/lib/lib.sh
#
# Lightweight logging & guard helpers for project scripts.
# Safe under: set -euo pipefail
#
# Optional:
#   LOG_TS=1     # prefix messages with [HH:MM:SS]
#   NO_COLOR=1   # disable ANSI color even when TTY

set -euo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)}"

# ==================================================================================== #
# LOGGING & ANSI OUTPUT
# ==================================================================================== #

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  BLUE=$'\033[0;34m'
  GREEN=$'\033[0;32m'
  YELLOW=$'\033[0;33m'
  RED=$'\033[0;31m'
  RESET=$'\033[0m'
else
  BLUE=''
  GREEN=''
  YELLOW=''
  RED=''
  RESET=''
fi

_ts() {
  if [[ -n "${LOG_TS:-}" ]]; then
    date +"%H:%M:%S"
  fi
  return 0
}

_prefix() {
  local t
  t="$(_ts)"
  if [[ -n "$t" ]]; then
    printf '[%s] ' "$t"
  fi
  return 0
}

info() {
  printf '%s' "${BLUE}"
  _prefix
  printf '→ %s%s\n' "$*" "${RESET}"
}

ok() {
  printf '%s' "${GREEN}"
  _prefix
  printf '✓ %s%s\n' "$*" "${RESET}"
}

warn() {
  printf '%s' "${YELLOW}"
  _prefix
  printf '! %s%s\n' "$*" "${RESET}"
}

err() {
  printf '%s' "${RED}" >&2
  _prefix >&2
  printf '✗ %s%s\n' "$*" "${RESET}" >&2
}

die() {
  err "$@"
  exit 1
}

# ==================================================================================== #
# GUARDS & HELPERS
# ==================================================================================== #

need_cmd() { command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"; }

require_file() {
  # Usage: require_file "path" ["message"]
  local path="${1:?path required}"
  local msg="${2:-Expected file not found: $path}"

  [[ -f "$path" ]] || die "$msg"
}

require_dir() {
  # Usage: require_dir "path" ["message"]
  local path="${1:?path required}"
  local msg="${2:-Expected directory not found: $path}"

  [[ -d "$path" ]] || die "$msg"
}

require_env() {
  # Usage: require_env VAR ["message"]
  local var="${1:?env var name required}"
  local msg="${2:-Expected environment variable not set: $var}"

  [[ -n "${!var:-}" ]] || die "$msg"
}

load_env_file() {
  # Usage: load_env_file ".env"
  local path="${1:?env file path required}"
  require_file "$path" "Refusing: $path not found. Create it (or copy from .env.example)."

  # Export all variables defined by sourcing the file.
  # Assumes env file is trusted and in KEY=VALUE form.
  set -a
  # shellcheck disable=SC1090
  source "$path"
  set +a
}

project_get() {
  # Usage: project_get "key"
  #
  # Reads a flat string key from .project.toml at repo root.
  # Supports lines of the form:
  #   key = "value"
  #
  # Returns the value on stdout.
  # Exits non-zero if the key is missing or malformed.

  local key="${1:?project key required}"
  local file="$ROOT_DIR/.project.toml"

  require_file "$file" "Refusing: missing project config: $file"

  local line
  line="$(
    sed -n -E \
      's/^[[:space:]]*'"$key"'[[:space:]]*=[[:space:]]*"([^"]*)"[[:space:]]*$/\1/p' \
      "$file" | head -n 1
  )"

  [[ -n "$line" ]] || die "Refusing: missing or invalid key '$key' in $file"

  printf '%s\n' "$line"
}

cleanup_file() {
  # Usage: cleanup_file "path"
  local path="${1:?path required}"
  rm -f "$path"
}
