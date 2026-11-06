#!/usr/bin/env bash
set -euo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
if [[ -z "$temporary" || ! -d "$temporary" ]]; then
  exit 1
fi
dirty_probe="$repository_root/.axiom-install-dirty-test-$$"
trap 'rm -rf -- "$temporary"; rm -f -- "$dirty_probe"' EXIT

export AXIOM_BIN_DIR="$temporary/bin"
export AXIOM_INSTALL_STATE_ROOT="$temporary/state"

case ":$PATH:" in
  *":$AXIOM_BIN_DIR:"*) exit 1 ;;
esac
# A legacy or foreign lingo executable beside the destination is preserved.
mkdir -p "$AXIOM_BIN_DIR"
chmod 700 "$AXIOM_BIN_DIR"
printf '#!/bin/sh\necho legacy\n' >"$AXIOM_BIN_DIR/lingo"
legacy_before=$(shasum -a 256 "$AXIOM_BIN_DIR/lingo" | awk '{print $1}')
first_stderr="$temporary/first.stderr"
first=$($repository_root/scripts/install-axiom.sh 2>"$first_stderr")
case "$first" in
  *'"result":"installed"'*'"pathConfigured":false'*) ;;
  *) exit 1 ;;
esac
if ! grep -Fq 'export PATH=' "$first_stderr" || ! grep -Fq "$AXIOM_BIN_DIR" "$first_stderr"; then
  exit 1
fi

if ! grep -Fq 'axiom is not on PATH' "$first_stderr" || grep -Fq 'lingo' "$first_stderr"; then
  exit 1
fi
if [[ ! -f "$AXIOM_BIN_DIR/axiom" || -L "$AXIOM_BIN_DIR/axiom" || ! -x "$AXIOM_BIN_DIR/axiom" ]]; then
  exit 1
fi
if [[ $(shasum -a 256 "$AXIOM_BIN_DIR/lingo" | awk '{print $1}') != "$legacy_before" ]]; then
  exit 1
fi
# The public contract is a real executable resolved from PATH by a fresh
# non-interactive shell with an empty environment: no alias or function.
if [[ $(cd / && env -i HOME="$temporary" PATH="$AXIOM_BIN_DIR:/usr/bin:/bin" bash --noprofile --norc -c 'type -t axiom') != file ]]; then
  exit 1
fi
version=$(cd / && env -i HOME="$temporary" PATH="$AXIOM_BIN_DIR:/usr/bin:/bin" sh -c 'axiom version')
if [[ $(PATH="$AXIOM_BIN_DIR:$PATH" command -v axiom) != "$AXIOM_BIN_DIR/axiom" ]]; then
  exit 1
fi
case "$version" in
  *'Succeeded (success)'*'Axiom build information'*'product: Axiom, version: development'*'sourceState: clean'*) ;;
  *) exit 1 ;;
esac

second_stderr="$temporary/second.stderr"
second=$(PATH="$AXIOM_BIN_DIR:$PATH" "$repository_root/scripts/install-axiom.sh" 2>"$second_stderr")
case "$second" in
  *'"result":"unchanged"'*'"pathConfigured":true'*) ;;
  *) exit 1 ;;
esac
if [[ -s "$second_stderr" ]]; then
  exit 1
fi

printf 'tampered\n' >>"$AXIOM_BIN_DIR/axiom"
before=$(shasum -a 256 "$AXIOM_BIN_DIR/axiom" | awk '{print $1}')
if "$repository_root/scripts/install-axiom.sh" >/dev/null 2>&1; then
  exit 1
fi
after=$(shasum -a 256 "$AXIOM_BIN_DIR/axiom" | awk '{print $1}')
if [[ "$before" != "$after" ]]; then
  exit 1
fi

export AXIOM_BIN_DIR="$temporary/dirty-bin"
export AXIOM_INSTALL_STATE_ROOT="$temporary/dirty-state"
: >"$dirty_probe"
dirty=$($repository_root/scripts/install-axiom.sh 2>"$temporary/dirty.stderr")
case "$dirty" in
  *'"result":"installed"'*'"dirty":true'*'"pathConfigured":false'*) ;;
  *) exit 1 ;;
esac
dirty_version=$(PATH="$AXIOM_BIN_DIR:$PATH" axiom --json version)
case "$dirty_version" in
  *'"version":"development"'*'"sourceState":"dirty"'*) ;;
  *) exit 1 ;;
esac
if ! grep -Fq 'dirty=true' "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt"; then
  exit 1
fi

rm -f -- "$dirty_probe"

# --replace-release: a verified release install is backed up, replaced by the
# checkout build with refreshed skills, and restored byte-for-byte.
export AXIOM_BIN_DIR="$temporary/release-bin"
export AXIOM_INSTALL_STATE_ROOT="$temporary/release-state"
mkdir -p "$AXIOM_BIN_DIR" "$AXIOM_INSTALL_STATE_ROOT" "$temporary/release-home" "$temporary/fake-runtime"
chmod 700 "$AXIOM_BIN_DIR" "$AXIOM_INSTALL_STATE_ROOT"
printf '#!/bin/sh\necho release\n' >"$AXIOM_BIN_DIR/axiom"
chmod 700 "$AXIOM_BIN_DIR/axiom"
release_sha=$(shasum -a 256 "$AXIOM_BIN_DIR/axiom" | awk '{print $1}')
printf 'formatVersion=1\ndestination=%s\nsha256=%s\nrelease=true\n' "$AXIOM_BIN_DIR/axiom" "$release_sha" >"$AXIOM_INSTALL_STATE_ROOT/installation.receipt"
printf '#!/bin/sh\nexit 0\n' >"$temporary/fake-runtime/claude"
chmod 700 "$temporary/fake-runtime/claude"
release_env=(env HOME="$temporary/release-home" GOMODCACHE="$(go env GOMODCACHE)" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" PATH="$temporary/fake-runtime:/usr/bin:/bin:$(dirname "$(command -v go)")" AXIOM_BIN_DIR="$AXIOM_BIN_DIR" AXIOM_INSTALL_STATE_ROOT="$AXIOM_INSTALL_STATE_ROOT")

if "$repository_root/scripts/install-axiom.sh" >/dev/null 2>&1; then
  exit 1 # plain mode still refuses a release-owned destination
fi
replaced=$("${release_env[@]}" "$repository_root/scripts/install-axiom.sh" --replace-release 2>/dev/null)
case "$replaced" in
  *'"result":"installed"'*) ;;
  *) exit 1 ;;
esac
case "$("$AXIOM_BIN_DIR/axiom" --json version)" in
  *'"version":"development"'*) ;;
  *) exit 1 ;;
esac
if [[ ! -f "$AXIOM_INSTALL_STATE_ROOT/release-backup/axiom" || -e "$AXIOM_INSTALL_STATE_ROOT/installation.receipt" || ! -f "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt" ]]; then
  exit 1
fi
if [[ ! -f "$temporary/release-home/.claude/skills/axiom-project/SKILL.md" ]]; then
  exit 1
fi
# A second dev install goes through the owned-source path; no second backup.
"${release_env[@]}" "$repository_root/scripts/install-axiom.sh" --replace-release >/dev/null 2>&1

# --restore-release refuses a drifted, unowned, or symlinked destination and
# leaves both the destination and the release backup untouched (CR-376-F2).
cp -p -- "$AXIOM_BIN_DIR/axiom" "$temporary/dev-axiom"
refuses_restore() {
  local before
  before=$(ls -l "$AXIOM_BIN_DIR/axiom" 2>/dev/null; shasum -a 256 "$AXIOM_BIN_DIR/axiom" 2>/dev/null || true)
  if "${release_env[@]}" "$repository_root/scripts/install-axiom.sh" --restore-release >/dev/null 2>&1; then
    exit 1
  fi
  if [[ "$(ls -l "$AXIOM_BIN_DIR/axiom" 2>/dev/null; shasum -a 256 "$AXIOM_BIN_DIR/axiom" 2>/dev/null || true)" != "$before" ]]; then
    exit 1
  fi
  if [[ $(shasum -a 256 "$AXIOM_INSTALL_STATE_ROOT/release-backup/axiom" | awk '{print $1}') != "$release_sha" || ! -f "$AXIOM_INSTALL_STATE_ROOT/release-backup/installation.receipt" ]]; then
    exit 1
  fi
}
printf 'drift\n' >>"$AXIOM_BIN_DIR/axiom"
refuses_restore
cp -p -- "$temporary/dev-axiom" "$AXIOM_BIN_DIR/axiom"
mv -- "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt" "$temporary/axiom.receipt"
refuses_restore
mv -- "$temporary/axiom.receipt" "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt"
rm -f -- "$AXIOM_BIN_DIR/axiom"
ln -s -- "$temporary/dev-axiom" "$AXIOM_BIN_DIR/axiom"
refuses_restore
rm -f -- "$AXIOM_BIN_DIR/axiom"
cp -p -- "$temporary/dev-axiom" "$AXIOM_BIN_DIR/axiom"

restored=$("${release_env[@]}" "$repository_root/scripts/install-axiom.sh" --restore-release 2>/dev/null || true)
case "$restored" in
  *'"result":"restored"'*) ;;
  *) exit 1 ;;
esac
if [[ $(shasum -a 256 "$AXIOM_BIN_DIR/axiom" | awk '{print $1}') != "$release_sha" || ! -f "$AXIOM_INSTALL_STATE_ROOT/installation.receipt" || -e "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt" || -e "$AXIOM_INSTALL_STATE_ROOT/release-backup" ]]; then
  exit 1
fi
if "${release_env[@]}" "$repository_root/scripts/install-axiom.sh" --restore-release >/dev/null 2>&1; then
  exit 1 # nothing left to restore
fi

# With the dev binary removed, restore still requires the dev receipt bound to
# this destination; a missing, symlinked, or foreign receipt keeps the backup (CR-376-F4).
"${release_env[@]}" "$repository_root/scripts/install-axiom.sh" --replace-release >/dev/null 2>&1
rm -f -- "$AXIOM_BIN_DIR/axiom"
mv -- "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt" "$temporary/axiom.receipt"
refuses_restore
ln -s -- "$temporary/axiom.receipt" "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt"
refuses_restore
rm -f -- "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt"
sed 's|^destination=.*|destination=/elsewhere/axiom|' "$temporary/axiom.receipt" >"$AXIOM_INSTALL_STATE_ROOT/axiom.receipt"
refuses_restore
mv -f -- "$temporary/axiom.receipt" "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt"
restored=$("${release_env[@]}" "$repository_root/scripts/install-axiom.sh" --restore-release 2>/dev/null || true)
case "$restored" in
  *'"result":"restored"'*) ;;
  *) exit 1 ;;
esac
if [[ $(shasum -a 256 "$AXIOM_BIN_DIR/axiom" | awk '{print $1}') != "$release_sha" || ! -f "$AXIOM_INSTALL_STATE_ROOT/installation.receipt" || -e "$AXIOM_INSTALL_STATE_ROOT/axiom.receipt" || -e "$AXIOM_INSTALL_STATE_ROOT/release-backup" ]]; then
  exit 1
fi

printf '%s\n' 'PASS: Axiom installer reports PATH setup, dirty metadata, idempotence, conflicts, and release replace/restore'
