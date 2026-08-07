#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-.}"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

pass() {
  printf 'PASS: %s\n' "$1"
}

[[ -d "$TARGET" ]] || fail "target directory does not exist: $TARGET"
ROOT="$(cd "$TARGET" && pwd -P)"

required_files=(
  ".gitignore"
  "AGENTS.md"
  "CHANGELOG.md"
  "README.md"
  "SECURITY.md"
  ".agents/policies/security.md"
  "docs/product/README.md"
  "docs/architecture/README.md"
  "docs/decisions/README.md"
  "docs/research/README.md"
  "docs/security/repository-security.md"
  "docs/development/getting-started.md"
  "docs/commands.md"
)

for relative in "${required_files[@]}"; do
  [[ -s "$ROOT/$relative" ]] || fail "required non-empty file is missing: $relative"
done

required_ignores=(
  ".env"
  ".env.*"
  "!.env.example"
  "*.pem"
  "*.key"
  "*.p12"
  "*.pfx"
  "id_rsa"
  "id_rsa.*"
  "id_ed25519"
  "id_ed25519.*"
  ".DS_Store"
  ".idea/"
  ".vscode/"
  "tmp/"
  ".temp/"
  ".cache/"
  "*.log"
  "coverage/"
  "dist/"
  "build/"
  "vendor/"
  "node_modules/"
)

for pattern in "${required_ignores[@]}"; do
  grep -Fxq -- "$pattern" "$ROOT/.gitignore" \
    || fail "required .gitignore pattern is missing: $pattern"
done

notion_url="https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979"

grep -Fq -- "$notion_url" "$ROOT/README.md" \
  || fail "Notion source is missing from README.md"
grep -Fq -- "$notion_url" "$ROOT/docs/product/README.md" \
  || fail "Notion source is missing from docs/product/README.md"

if find "$ROOT" -path "$ROOT/.git" -prune -o \( -name 'CLAUDE.md' -o -name '.claude' \) -print -quit | grep -q .; then
  fail "Claude-specific artifact found"
fi

if find "$ROOT/docs" -type d -empty -print -quit | grep -q .; then
  fail "empty documentation directory found"
fi

"$ROOT/scripts/validate-agent-package.sh" "$ROOT"
"$ROOT/scripts/test-validate-agent-package.sh"
"$ROOT/scripts/test-check-sensitive-files.sh"
"$ROOT/scripts/check-sensitive-files.sh" "$ROOT"
git -C "$ROOT" diff --check

pass "repository bootstrap validation passed"
