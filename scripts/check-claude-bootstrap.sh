#!/usr/bin/env bash
# Closed check for the approved maintainer-runtime bootstrap (see AGENTS.md,
# "Maintainer runtimes", and ADR-0014).
#
# Default (generated packages): the only allowed Claude artifact is a regular
# root CLAUDE.md whose entire content is "@AGENTS.md" with an optional final
# newline.
#
# --skill-adapters (this repository): additionally allows exactly one Claude
# discovery adapter per canonical maintainer skill, the relative symlink
# .claude/skills/<skill> -> ../../.agents/skills/<skill>. Anything else under
# .claude/ fails.
set -euo pipefail

ADAPTERS=0
if [[ "${1:-}" == "--skill-adapters" ]]; then
  ADAPTERS=1
  shift
fi
TARGET="${1:-.}"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -d "$TARGET" ]] || fail "target directory does not exist: $TARGET"
ROOT="$(cd "$TARGET" && pwd -P)"
BOOTSTRAP="$ROOT/CLAUDE.md"
CLAUDE_DIR="$ROOT/.claude"
ADAPTER_DIR="$CLAUDE_DIR/skills"
CANONICAL_DIR="$ROOT/.agents/skills"

# Any Claude instruction or configuration entry other than the root bootstrap
# (and, with adapters, the root .claude directory checked below), of any type
# and in any letter case, is unapproved. find does not follow the adapter
# symlinks, so canonical skill content is only scanned at its canonical path.
while IFS= read -r -d '' entry; do
  [[ "$entry" == "$BOOTSTRAP" ]] && continue
  [[ "$ADAPTERS" == 1 && "$entry" == "$CLAUDE_DIR" ]] && continue
  fail "unapproved Claude artifact: ${entry#"$ROOT/"}"
done < <(find "$ROOT" -path "$ROOT/.git" -prune -o \
  \( -iname 'CLAUDE.md' -o -iname 'CLAUDE.local.md' -o -iname '.claude' \) -print0)

if [[ -e "$BOOTSTRAP" || -L "$BOOTSTRAP" ]]; then
  [[ -f "$BOOTSTRAP" && ! -L "$BOOTSTRAP" ]] || fail "CLAUDE.md must be a regular file"
  cmp -s "$BOOTSTRAP" <(printf '@AGENTS.md\n') || cmp -s "$BOOTSTRAP" <(printf '@AGENTS.md') \
    || fail "CLAUDE.md must contain exactly @AGENTS.md"
  [[ -f "$ROOT/AGENTS.md" && ! -L "$ROOT/AGENTS.md" ]] || fail "CLAUDE.md requires a regular AGENTS.md"
fi

[[ "$ADAPTERS" == 1 ]] || exit 0

real_dir() {
  [[ -d "$1" && ! -L "$1" ]]
}

# Prints the frontmatter name of a SKILL.md, or nothing.
skill_name() {
  awk 'NR == 1 && $0 != "---" { exit }
    NR > 1 && $0 == "---" { exit }
    NR > 1 && /^name:[ \t]*/ { sub(/^name:[ \t]*/, ""); gsub(/^["\047]|["\047][ \t]*$/, ""); print; exit }' "$1"
}

# Canonical maintainer skills: real directories with a regular SKILL.md whose
# name matches the directory.
canonical=()
if [[ -e "$ROOT/.agents" || -L "$ROOT/.agents" ]]; then
  real_dir "$ROOT/.agents" || fail ".agents must be a real directory"
fi
if [[ -e "$CANONICAL_DIR" || -L "$CANONICAL_DIR" ]]; then
  real_dir "$CANONICAL_DIR" || fail ".agents/skills must be a real directory"
  while IFS= read -r -d '' skill; do
    name="${skill##*/}"
    real_dir "$skill" || fail "canonical skill must be a real directory: .agents/skills/$name"
    [[ -f "$skill/SKILL.md" && ! -L "$skill/SKILL.md" ]] \
      || fail "canonical skill must have a regular SKILL.md: .agents/skills/$name"
    [[ "$(skill_name "$skill/SKILL.md")" == "$name" ]] \
      || fail "canonical skill name must match its directory: .agents/skills/$name"
    canonical+=("$name")
  done < <(find "$CANONICAL_DIR" -mindepth 1 -maxdepth 1 -print0)
fi

is_canonical() {
  local candidate
  for candidate in ${canonical[@]+"${canonical[@]}"}; do
    [[ "$candidate" == "$1" ]] && return 0
  done
  return 1
}

if [[ -e "$CLAUDE_DIR" || -L "$CLAUDE_DIR" ]]; then
  real_dir "$CLAUDE_DIR" || fail ".claude must be a real directory"
  while IFS= read -r -d '' entry; do
    [[ "$entry" == "$ADAPTER_DIR" ]] || fail "unapproved Claude artifact: ${entry#"$ROOT/"}"
  done < <(find "$CLAUDE_DIR" -mindepth 1 -maxdepth 1 -print0)
  real_dir "$ADAPTER_DIR" || fail ".claude may only contain a real skills directory"

  while IFS= read -r -d '' adapter; do
    name="${adapter##*/}"
    relative="${adapter#"$ROOT/"}"
    [[ "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || fail "unapproved Claude adapter name: $relative"
    [[ -L "$adapter" ]] \
      || fail "Claude adapter must be a symlink to its canonical skill, not a copy: $relative"
    [[ "$(readlink "$adapter")" == "../../.agents/skills/$name" ]] \
      || fail "Claude adapter must point to ../../.agents/skills/$name: $relative"
    is_canonical "$name" || fail "Claude adapter has no canonical skill: $relative"
    [[ "$(cd -P "$adapter" 2>/dev/null && pwd -P)" == "$CANONICAL_DIR/$name" ]] \
      || fail "Claude adapter does not resolve to its canonical skill: $relative"
  done < <(find "$ADAPTER_DIR" -mindepth 1 -maxdepth 1 -print0)
fi

for name in ${canonical[@]+"${canonical[@]}"}; do
  [[ -L "$ADAPTER_DIR/$name" ]] || fail "canonical skill has no Claude discovery adapter: .claude/skills/$name"
done

# A maintainer skill has one implementation: no other SKILL.md may declare a
# canonical maintainer skill name.
while IFS= read -r -d '' file; do
  if [[ "$file" == "$CANONICAL_DIR/"*/SKILL.md && "${file#"$CANONICAL_DIR/"}" != */*/* ]]; then
    continue
  fi
  [[ ! -L "$file" ]] || fail "SKILL.md must not be a symlink: ${file#"$ROOT/"}"
  [[ -f "$file" ]] || continue
  declared="$(skill_name "$file")"
  if [[ -n "$declared" ]] && is_canonical "$declared"; then
    fail "duplicate implementation of canonical skill $declared: ${file#"$ROOT/"}"
  fi
done < <(find "$ROOT" -path "$ROOT/.git" -prune -o -iname 'SKILL.md' -print0)

exit 0
