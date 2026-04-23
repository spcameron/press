#!/usr/bin/env bash
# shellcheck shell=bash
#
# scripts/lib/git.sh

set -euo pipefail

source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

require_clean() {
  need_cmd git

  local status
  status="$(git status --porcelain)"

  if [[ -n "$status" ]]; then
    err "Refusing: working tree is dirty."
    printf '\n%s\n\n' "$status" >&2
    warn "Commit or stash changes first."
    exit 1
  fi
}

require_upstream() {
  need_cmd git

  local up
  up="$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)"

  if [[ -z "$up" ]]; then
    err "Refusing: no upstream set for this branch."
    warn "Run 'push-upstream' first."
    exit 1
  fi
}

require_on_main() {
  need_cmd git

  local branch
  branch="$(git rev-parse --abbrev-ref HEAD)"

  if [[ "$branch" == "HEAD" ]]; then
    err "Refusing: detached HEAD (checkout main)."
    exit 1
  fi

  if [[ "$branch" != "main" ]]; then
    err "Refusing: not on main branch (current: '$branch')."
    exit 1
  fi
}

require_on_feature() {
  need_cmd git

  local branch allowed
  allowed="feature|fix|refactor|chore|docs"

  branch="$(git rev-parse --abbrev-ref HEAD)"

  if [[ "$branch" == "HEAD" ]]; then
    err "Refusing: detached HEAD (checkout a branch)."
    exit 1
  fi

  case "$branch" in
  feature/* | fix/* | refactor/* | chore/* | docs/*) : ;;
  main)
    err "Refusing: on main branch."
    warn "Use a work branch ($allowed)."
    exit 1
    ;;
  *)
    err "Refusing: branch '$branch' has an unapproved prefix."
    warn "Use a work branch ($allowed)."
    exit 1
    ;;
  esac
}

require_main_matches_origin() {
  need_cmd git
  require_on_main

  git fetch origin
  if [[ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]]; then
    err "Refusing: local HEAD does not match 'origin/main'."
    warn "Run 'sync-main' first."
    exit 1
  fi
}
