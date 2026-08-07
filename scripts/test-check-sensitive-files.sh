#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
CHECK="$ROOT/scripts/check-sensitive-files.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

new_fixture() {
  local fixture="$1"
  mkdir -p "$fixture"
  git -C "$fixture" init --quiet
  git -C "$fixture" config user.name "Axiom Test"
  git -C "$fixture" config user.email "axiom-test@example.invalid"
  printf '.env\n.env.*\n!.env.example\n' > "$fixture/.gitignore"
}

expect_pass() {
  local description="$1"
  shift

  "$@" >/dev/null 2>&1 || fail "$description"
}

expect_fail() {
  local description="$1"
  shift

  if "$@" >/dev/null 2>&1; then
    fail "$description"
  fi
}

[[ -x "$CHECK" ]] || fail "sensitive-file checker is missing or not executable"

TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/axiom-sensitive-files.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

SAFE_REPO="$TEST_ROOT/safe"
new_fixture "$SAFE_REPO"
printf 'EXAMPLE_VALUE=replace-me\n' > "$SAFE_REPO/.env.example"
expect_pass ".env.example should be allowed" "$CHECK" "$SAFE_REPO"

ENV_REPO="$TEST_ROOT/env"
new_fixture "$ENV_REPO"
printf 'REAL_VALUE=do-not-commit\n' > "$ENV_REPO/.env.local"
git -C "$ENV_REPO" add --force .env.local
expect_fail "staged .env.local should be rejected" "$CHECK" --staged "$ENV_REPO"

KEY_REPO="$TEST_ROOT/key"
new_fixture "$KEY_REPO"
printf '%s%s\n%s\n' '-----BEGIN ' 'PRIVATE KEY-----' 'test-only' > "$KEY_REPO/notes.txt"
expect_fail "private-key content should be rejected" "$CHECK" "$KEY_REPO"

TOKEN_REPO="$TEST_ROOT/token"
new_fixture "$TOKEN_REPO"
printf '%s%s\n' 'ghp_' 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' > "$TOKEN_REPO/token.txt"
expect_fail "GitHub token pattern should be rejected" "$CHECK" "$TOKEN_REPO"

STAGED_REPO="$TEST_ROOT/staged"
new_fixture "$STAGED_REPO"
printf 'safe\n' > "$STAGED_REPO/safe.txt"
printf 'ignored locally\n' > "$STAGED_REPO/.env.local"
git -C "$STAGED_REPO" add .gitignore safe.txt
expect_pass "staged mode should ignore unstaged ignored files" "$CHECK" --staged "$STAGED_REPO"

INDEX_REPO="$TEST_ROOT/index"
new_fixture "$INDEX_REPO"
printf '%s%s\n' 'ghp_' 'BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB' > "$INDEX_REPO/staged.txt"
git -C "$INDEX_REPO" add staged.txt
printf 'safe worktree content\n' > "$INDEX_REPO/staged.txt"
expect_fail "staged mode should scan index content, not worktree content" \
  "$CHECK" --staged "$INDEX_REPO"

SAFE_DIRECTORY="$TEST_ROOT/safe-directory"
mkdir -p "$SAFE_DIRECTORY"
printf 'EXAMPLE_VALUE=replace-me\n' > "$SAFE_DIRECTORY/.env.example"
expect_pass "directory mode should work without a Git repository" \
  "$CHECK" --directory "$SAFE_DIRECTORY"

SECRET_DIRECTORY="$TEST_ROOT/secret-directory"
mkdir -p "$SECRET_DIRECTORY"
printf 'REAL_VALUE=do-not-package\n' > "$SECRET_DIRECTORY/.env.production"
expect_fail "directory mode should reject sensitive filenames" \
  "$CHECK" --directory "$SECRET_DIRECTORY"

printf 'PASS: sensitive-file checker behavior is valid\n'
