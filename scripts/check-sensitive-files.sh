#!/usr/bin/env bash
set -euo pipefail

MODE="worktree"

case "${1:-}" in
  --staged)
    MODE="staged"
    shift
    ;;
  --directory)
    MODE="directory"
    shift
    ;;
esac

TARGET="${1:-.}"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -d "$TARGET" ]] || fail "target directory does not exist: $TARGET"

ROOT="$(cd "$TARGET" && pwd -P)"

if [[ "$MODE" != "directory" ]]; then
  git -C "$ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1 \
    || fail "target is not inside a Git worktree: $ROOT"
fi

cd "$ROOT"

list_candidates() {
  if [[ "$MODE" == "directory" ]]; then
    find . -path './.git' -prune -o -type f -print0
    return
  fi

  if [[ "$MODE" == "staged" ]]; then
    git diff --cached --name-only --diff-filter=ACMR -z
    return
  fi

  git ls-files --cached --others --exclude-standard -z
}

is_sensitive_path() {
  local relative="$1"
  local base="${relative##*/}"

  if [[ "$base" == ".env.example" ]]; then
    return 1
  fi

  case "$base" in
    .env|.env.*|*.pem|*.key|*.p12|*.pfx|id_rsa|id_rsa.*|id_ed25519|id_ed25519.*|*.dump|*.sql.gz|*.sql.bz2|*.sql.xz|*.db|*.sqlite|*.sqlite3|*.log)
      return 0
      ;;
  esac

  case "$relative" in
    .aws/credentials|*/.aws/credentials)
      return 0
      ;;
  esac

  return 1
}

contains_sensitive_content() {
  local relative="$1"
  local private_key_pattern='-----BEGIN ([A-Z0-9]+ )?PRIVATE KEY-----'
  local github_token_pattern='(gh[pousr]_[A-Za-z0-9_]{36,255}|github_pat_[A-Za-z0-9_]{20,255})'
  local aws_access_key_pattern='(AKIA|ASIA)[A-Z0-9]{16}'
  local sensitive_pattern="($private_key_pattern)|($github_token_pattern)|($aws_access_key_pattern)"

  if [[ "$MODE" == "staged" ]]; then
    git show ":$relative" | LC_ALL=C grep -IE -- "$sensitive_pattern" >/dev/null
    return
  fi

  LC_ALL=C grep -IqE -- "$sensitive_pattern" "$relative"
}

failures=0

while IFS= read -r -d '' relative; do
  if [[ "$MODE" == "worktree" && ! -f "$relative" ]]; then
    continue
  fi

  if is_sensitive_path "$relative"; then
    printf 'ERROR: prohibited sensitive-file path: %s\n' "$relative" >&2
    failures=$((failures + 1))
    continue
  fi

  if contains_sensitive_content "$relative"; then
    printf 'ERROR: high-confidence secret pattern found in: %s\n' "$relative" >&2
    failures=$((failures + 1))
  fi
done < <(list_candidates)

if ((failures > 0)); then
  fail "sensitive-file checks found $failures issue(s)"
fi

printf 'PASS: sensitive-file checks passed (%s)\n' "$MODE"
