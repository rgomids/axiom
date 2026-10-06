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
  "docs/README.pt-BR.md"
  "SECURITY.md"
  ".agents/policies/security.md"
  ".agents/policies/repository-automation.md"
  "scripts/automation-registry.json"
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

while IFS= read -r -d '' root_readme; do
  [[ "${root_readme#"$ROOT/"}" == "README.md" ]] \
    || fail "only README.md is allowed at the repository root: ${root_readme#"$ROOT/"}"
done < <(find "$ROOT" -maxdepth 1 -type f -name 'README*.md' -print0)

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

wiki_url="https://github.com/rgomids/axiom/wiki"

grep -Fq -- "$wiki_url" "$ROOT/README.md" \
  || fail "Wiki source is missing from README.md"
grep -Fq -- "$wiki_url" "$ROOT/docs/README.pt-BR.md" \
  || fail "Wiki source is missing from docs/README.pt-BR.md"
grep -Fq -- 'href="docs/README.pt-BR.md"' "$ROOT/README.md" \
  || fail "README.md is missing the Brazilian Portuguese navigation link"
grep -Fq -- 'href="../README.md"' "$ROOT/docs/README.pt-BR.md" \
  || fail "docs/README.pt-BR.md is missing the English navigation link"
grep -Fq -- "$wiki_url" "$ROOT/docs/product/README.md" \
  || fail "Wiki source is missing from docs/product/README.md"

"$ROOT/scripts/check-claude-bootstrap.sh" --skill-adapters "$ROOT"

if find "$ROOT/docs" -type d -empty -print -quit | grep -q .; then
  fail "empty documentation directory found"
fi

"$ROOT/scripts/validate-agent-package.sh" --maintainer-harness "$ROOT"
"$ROOT/scripts/test-validate-agent-package.sh"
"$ROOT/scripts/test-check-claude-bootstrap.sh"
"$ROOT/scripts/test-maintainer-agent.sh"
python3 "$ROOT/scripts/test-check-automation-registry.py"
python3 "$ROOT/scripts/check-automation-registry.py" "$ROOT"
python3 "$ROOT/scripts/test-issue-label-policy.py"
python3 "$ROOT/scripts/test-gate-evidence.py"
python3 "$ROOT/scripts/test-prepared-upgrade-candidate.py"
python3 "$ROOT/scripts/test-check-adr-governance.py"
python3 "$ROOT/scripts/check-adr-governance.py" "$ROOT"
"$ROOT/scripts/test-check-sensitive-files.sh"
"$ROOT/scripts/check-sensitive-files.sh" "$ROOT"
git -C "$ROOT" diff --check

pass "repository bootstrap validation passed"
