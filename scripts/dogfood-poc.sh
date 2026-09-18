#!/usr/bin/env bash
set -euo pipefail

temporary=$(mktemp -d)
if [[ -z "$temporary" || ! -d "$temporary" ]]; then
  exit 1
fi
trap 'rm -rf -- "$temporary"' EXIT

export LINGO_PROJECTS_ROOT="$temporary/projects"
export LINGO_STATE_ROOT="$temporary/state"
go build -o "$temporary/lingo" ./cmd/lingo

"$temporary/lingo" project init --slug axiom-poc-hardening --name "Axiom POC persistence hardening"
"$temporary/lingo" project validate --slug axiom-poc-hardening
"$temporary/lingo" project reopen --slug axiom-poc-hardening

cp "$LINGO_PROJECTS_ROOT/axiom-poc-hardening/axiom.yaml" "$temporary/before-install.yaml"
"$temporary/lingo" project install --source "$LINGO_PROJECTS_ROOT/axiom-poc-hardening"
cmp "$temporary/before-install.yaml" "$LINGO_PROJECTS_ROOT/axiom-poc-hardening/axiom.yaml"
"$temporary/lingo" project reopen --slug axiom-poc-hardening
"$temporary/lingo" project update --slug axiom-poc-hardening --name "Axiom POC hardening verified"
"$temporary/lingo" project validate --slug axiom-poc-hardening

record_count=$(find "$LINGO_STATE_ROOT/projects" -name installation.json -type f | wc -l | tr -d ' ')
if [[ "$record_count" != 1 ]]; then
  exit 1
fi
if "$temporary/lingo" project reopen --slug axiom-poc-hardening; then
  exit 1
fi
printf '%s\n' 'PASS: portable bytes unchanged by install; one ID-addressed local record; stale local state detected after update'
