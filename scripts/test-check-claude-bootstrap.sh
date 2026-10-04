#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
CHECK="$ROOT/scripts/check-claude-bootstrap.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -x "$CHECK" ]] || fail "Claude bootstrap check is missing or not executable"

TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/axiom-claude-bootstrap.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT
count=0

new_root() {
  count=$((count + 1))
  root="$TEST_ROOT/case-$count"
  mkdir -p "$root/docs"
  printf '# Agent policy\n' > "$root/AGENTS.md"
}

expect_pass() {
  "$CHECK" "$2" >/dev/null 2>&1 || fail "$1 should pass"
}

expect_fail() {
  if "$CHECK" "$2" >/dev/null 2>&1; then
    fail "$1 should fail"
  fi
}

new_root
expect_pass "absent CLAUDE.md" "$root"

new_root
printf '@AGENTS.md\n' > "$root/CLAUDE.md"
expect_pass "root exact pointer" "$root"

new_root
printf '@AGENTS.md' > "$root/CLAUDE.md"
expect_pass "root exact pointer without final newline" "$root"

new_root
printf '@AGENTS.md\nAlways prefer Claude-only rules.\n' > "$root/CLAUDE.md"
expect_fail "root pointer plus own instructions" "$root"

new_root
printf '@AGENTS.md\n\n' > "$root/CLAUDE.md"
expect_fail "root pointer plus extra blank line" "$root"

new_root
printf '@AGENTS.md\r\n' > "$root/CLAUDE.md"
expect_fail "root pointer with CRLF" "$root"

new_root
printf '@README.md\n' > "$root/CLAUDE.md"
expect_fail "pointer to a different target" "$root"

new_root
printf '@docs/AGENTS.md\n' > "$root/CLAUDE.md"
expect_fail "pointer to a nested target" "$root"

new_root
printf 'Follow AGENTS.md.\n' > "$root/CLAUDE.md"
expect_fail "prose instead of the pointer" "$root"

new_root
printf '@AGENTS.md\n' > "$root/CLAUDE.md"
printf '@AGENTS.md\n' > "$root/docs/CLAUDE.md"
expect_fail "nested CLAUDE.md" "$root"

new_root
printf '@AGENTS.md\n' > "$root/claude.md"
expect_fail "differently cased root claude.md" "$root"

new_root
printf '@AGENTS.md\n' > "$root/CLAUDE.local.md"
expect_fail "CLAUDE.local.md" "$root"

new_root
printf '@AGENTS.md\n' > "$root/CLAUDE.md"
mkdir "$root/.claude"
expect_fail ".claude directory" "$root"

new_root
printf '{}\n' > "$root/.claude"
expect_fail ".claude file" "$root"

new_root
mkdir -p "$root/docs/.claude"
expect_fail "nested .claude directory" "$root"

new_root
ln -s AGENTS.md "$root/CLAUDE.md"
expect_fail "CLAUDE.md symlink to AGENTS.md" "$root"

new_root
printf '@AGENTS.md\n' > "$TEST_ROOT/outside-claude.md"
ln -s "$TEST_ROOT/outside-claude.md" "$root/CLAUDE.md"
expect_fail "CLAUDE.md symlink to an exact pointer outside the root" "$root"

new_root
mkdir "$root/CLAUDE.md"
expect_fail "CLAUDE.md directory" "$root"

new_root
printf '@AGENTS.md\n' > "$root/CLAUDE.md"
rm "$root/AGENTS.md"
expect_fail "pointer without AGENTS.md" "$root"

new_root
printf '@AGENTS.md\n' > "$root/CLAUDE.md"
mv "$root/AGENTS.md" "$root/policy.md"
ln -s policy.md "$root/AGENTS.md"
expect_fail "pointer to a symlinked AGENTS.md" "$root"

# --skill-adapters: the closed allowlist of Claude discovery adapters (ADR-0014).

expect_pass_adapters() {
  "$CHECK" --skill-adapters "$2" >/dev/null 2>&1 || fail "$1 should pass with adapters"
}

expect_fail_adapters() {
  if "$CHECK" --skill-adapters "$2" >/dev/null 2>&1; then
    fail "$1 should fail with adapters"
  fi
}

new_skill() {
  mkdir -p "$root/.agents/skills/$1"
  printf '%s\n' '---' "name: $1" 'description: Test skill.' '---' "# $1" \
    > "$root/.agents/skills/$1/SKILL.md"
}

new_harness() {
  new_root
  printf '@AGENTS.md\n' > "$root/CLAUDE.md"
  new_skill alpha
  new_skill beta-gamma
  mkdir -p "$root/.claude/skills"
  ln -s ../../.agents/skills/alpha "$root/.claude/skills/alpha"
  ln -s ../../.agents/skills/beta-gamma "$root/.claude/skills/beta-gamma"
}

new_harness
expect_pass_adapters "one adapter per canonical skill" "$root"
expect_fail "adapters without --skill-adapters (generated package mode)" "$root"

new_root
expect_pass_adapters "no canonical skills and no adapters" "$root"

new_root
new_skill alpha
expect_fail_adapters "canonical skill without an adapter" "$root"

new_harness
rm "$root/.claude/skills/beta-gamma"
expect_fail_adapters "missing adapter for one canonical skill" "$root"

new_harness
printf '@AGENTS.md\nClaude-only rule.\n' > "$root/CLAUDE.md"
expect_fail_adapters "adapters with a CLAUDE.md that is not exactly @AGENTS.md" "$root"

new_harness
printf '{}\n' > "$root/.claude/settings.json"
expect_fail_adapters ".claude/settings.json" "$root"

new_harness
printf '{}\n' > "$root/.claude/settings.local.json"
expect_fail_adapters ".claude/settings.local.json" "$root"

new_harness
mkdir "$root/.claude/agents"
expect_fail_adapters ".claude/agents directory" "$root"

new_harness
mkdir "$root/.claude/commands"
expect_fail_adapters ".claude/commands directory" "$root"

new_harness
printf 'note\n' > "$root/.claude/skills/README.md"
expect_fail_adapters "unexpected file in .claude/skills" "$root"

new_harness
printf 'note\n' > "$root/.claude/skills/.hidden"
expect_fail_adapters "hidden file in .claude/skills" "$root"

new_harness
mkdir "$root/.claude/skills/delta"
expect_fail_adapters "unexpected directory in .claude/skills" "$root"

new_harness
printf '@AGENTS.md\n' > "$root/.claude/CLAUDE.md"
expect_fail_adapters ".claude/CLAUDE.md" "$root"

new_harness
rm "$root/.claude/skills/alpha"
mkdir "$root/.claude/skills/alpha"
cp "$root/.agents/skills/alpha/SKILL.md" "$root/.claude/skills/alpha/SKILL.md"
expect_fail_adapters "copied SKILL.md instead of an adapter" "$root"

new_harness
mkdir -p "$root/docs/alpha-copy"
cp "$root/.agents/skills/alpha/SKILL.md" "$root/docs/alpha-copy/SKILL.md"
expect_fail_adapters "duplicate SKILL.md declaring a canonical name elsewhere" "$root"

new_harness
ln -s ../../.agents/skills/alpha/SKILL.md "$root/docs/SKILL.md"
expect_fail_adapters "symlinked SKILL.md outside the adapters" "$root"

new_harness
rm "$root/.claude/skills/alpha"
ln -s "$root/.agents/skills/alpha" "$root/.claude/skills/alpha"
expect_fail_adapters "absolute adapter target" "$root"

new_harness
rm "$root/.claude/skills/alpha"
ln -s ../../.agents/skills/alpha/ "$root/.claude/skills/alpha"
expect_fail_adapters "non-exact adapter target (trailing slash)" "$root"

new_harness
rm "$root/.claude/skills/alpha"
ln -s ../../.agents/skills/../skills/alpha "$root/.claude/skills/alpha"
expect_fail_adapters "adapter target with traversal" "$root"

new_harness
rm "$root/.claude/skills/alpha"
ln -s ../../.agents/skills/beta-gamma "$root/.claude/skills/alpha"
expect_fail_adapters "adapter pointing to a different canonical skill" "$root"

new_harness
rm "$root/.claude/skills/alpha"
ln -s ../../.agents/skills/alpha/SKILL.md "$root/.claude/skills/alpha"
expect_fail_adapters "adapter pointing to a file" "$root"

new_harness
rm "$root/.claude/skills/alpha"
mkdir -p "$TEST_ROOT/outside-$count/skills/alpha"
ln -s "../../../outside-$count/skills/alpha" "$root/.claude/skills/alpha"
expect_fail_adapters "adapter pointing outside the repository" "$root"

new_harness
ln -s ../../.agents/skills/omega "$root/.claude/skills/omega"
expect_fail_adapters "broken adapter without a canonical target" "$root"

new_harness
rm "$root/.claude/skills/alpha"
ln -s ../../.agents/skills/ALPHA "$root/.claude/skills/ALPHA"
expect_fail_adapters "adapter with an unapproved name" "$root"

new_harness
sed -i.bak 's/^name: alpha$/name: other/' "$root/.agents/skills/alpha/SKILL.md"
rm "$root/.agents/skills/alpha/SKILL.md.bak"
expect_fail_adapters "canonical name mismatching its directory" "$root"

new_harness
rm "$root/.agents/skills/alpha/SKILL.md"
expect_fail_adapters "canonical skill without SKILL.md" "$root"

new_harness
mv "$root/.agents/skills/alpha" "$TEST_ROOT/moved-alpha-$count"
ln -s "../../../moved-alpha-$count" "$root/.agents/skills/alpha"
expect_fail_adapters "canonical skill directory replaced by a symlink" "$root"

new_harness
mv "$root/.agents" "$TEST_ROOT/moved-agents-$count"
ln -s "$TEST_ROOT/moved-agents-$count" "$root/.agents"
expect_fail_adapters "symlinked .agents directory" "$root"

new_harness
mv "$root/.claude/skills" "$root/claude-skills"
ln -s ../claude-skills "$root/.claude/skills"
expect_fail_adapters "symlinked .claude/skills directory" "$root"

new_harness
mv "$root/.claude" "$root/claude-dir"
ln -s claude-dir "$root/.claude"
expect_fail_adapters "symlinked .claude directory" "$root"

new_harness
mkdir -p "$root/docs/.claude/skills"
expect_fail_adapters "nested .claude directory with adapters" "$root"

new_harness
rm -rf "$root/.claude/skills"
expect_fail_adapters ".claude without a skills directory" "$root"

# Both repository validators must delegate to this closed check: the
# repository with adapters, generated packages without.
grep -Fq '"$ROOT/scripts/check-claude-bootstrap.sh" --skill-adapters "$ROOT"' "$ROOT/scripts/validate-repository.sh" \
  || fail "validate-repository.sh does not run the Claude adapter check"
grep -Fq '"$SCRIPT_DIR/check-claude-bootstrap.sh" --skill-adapters "$TARGET"' "$ROOT/scripts/validate-agent-package.sh" \
  || fail "validate-agent-package.sh does not run the Claude adapter check in maintainer-harness mode"
grep -Fq '"$SCRIPT_DIR/check-claude-bootstrap.sh" "$TARGET"' "$ROOT/scripts/validate-agent-package.sh" \
  || fail "validate-agent-package.sh does not run the Claude bootstrap check"
grep -Fq '"$ROOT/scripts/validate-agent-package.sh" --maintainer-harness "$ROOT"' "$ROOT/scripts/validate-repository.sh" \
  || fail "validate-repository.sh does not validate the maintainer harness"

printf 'PASS: Claude bootstrap check behavior is valid (%s cases)\n' "$count"
