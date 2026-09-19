#!/usr/bin/env bash
set -euo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
if [[ -z "$temporary" || ! -d "$temporary" ]]; then
  exit 1
fi
trap 'rm -rf -- "$temporary"' EXIT

export AXIOM_BIN_DIR="$temporary/bin"
export AXIOM_INSTALL_STATE_ROOT="$temporary/state"

first=$($repository_root/scripts/install-axiom.sh)
case "$first" in
  *'"result":"installed"'*) ;;
  *) exit 1 ;;
esac

version=$(cd / && PATH="$AXIOM_BIN_DIR:$PATH" lingo version)
case "$version" in
  *'"product":"Axiom"'*'"binary":"lingo"'*'"source":"https://github.com/rgomids/axiom"'*) ;;
  *) exit 1 ;;
esac

second=$($repository_root/scripts/install-axiom.sh)
case "$second" in
  *'"result":"unchanged"'*) ;;
  *) exit 1 ;;
esac

printf 'tampered\n' >>"$AXIOM_BIN_DIR/lingo"
before=$(shasum -a 256 "$AXIOM_BIN_DIR/lingo" | awk '{print $1}')
if "$repository_root/scripts/install-axiom.sh" >/dev/null 2>&1; then
  exit 1
fi
after=$(shasum -a 256 "$AXIOM_BIN_DIR/lingo" | awk '{print $1}')
if [[ "$before" != "$after" ]]; then
  exit 1
fi

printf '%s\n' 'PASS: Axiom installer is idempotent, globally invokable, and conflict-safe'
