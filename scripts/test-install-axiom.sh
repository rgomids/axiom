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
first_stderr="$temporary/first.stderr"
first=$($repository_root/scripts/install-axiom.sh 2>"$first_stderr")
case "$first" in
  *'"result":"installed"'*'"pathConfigured":false'*) ;;
  *) exit 1 ;;
esac
if ! grep -Fq 'export PATH=' "$first_stderr" || ! grep -Fq "$AXIOM_BIN_DIR" "$first_stderr"; then
  exit 1
fi

version=$(cd / && PATH="$AXIOM_BIN_DIR:$PATH" lingo version)
if [[ $(PATH="$AXIOM_BIN_DIR:$PATH" command -v lingo) != "$AXIOM_BIN_DIR/lingo" ]]; then
  exit 1
fi
case "$version" in
  *'"product":"Axiom"'*'"binary":"lingo"'*'"source":"https://github.com/rgomids/axiom"'*) ;;
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

printf 'tampered\n' >>"$AXIOM_BIN_DIR/lingo"
before=$(shasum -a 256 "$AXIOM_BIN_DIR/lingo" | awk '{print $1}')
if "$repository_root/scripts/install-axiom.sh" >/dev/null 2>&1; then
  exit 1
fi
after=$(shasum -a 256 "$AXIOM_BIN_DIR/lingo" | awk '{print $1}')
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
dirty_version=$(PATH="$AXIOM_BIN_DIR:$PATH" lingo version)
case "$dirty_version" in
  *'"dirty":true'*'"dirtyKnown":true'*) ;;
  *) exit 1 ;;
esac
if ! grep -Fq 'dirty=true' "$AXIOM_INSTALL_STATE_ROOT/lingo.receipt"; then
  exit 1
fi

printf '%s\n' 'PASS: Axiom installer reports PATH setup, dirty metadata, idempotence, and conflicts'
