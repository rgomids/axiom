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
portable_before_sha=$(shasum -a 256 "$temporary/before-install.yaml" | cut -d ' ' -f 1)
"$temporary/lingo" project install --source "$LINGO_PROJECTS_ROOT/axiom-poc-hardening"
cmp "$temporary/before-install.yaml" "$LINGO_PROJECTS_ROOT/axiom-poc-hardening/axiom.yaml"
portable_after_install_sha=$(shasum -a 256 "$LINGO_PROJECTS_ROOT/axiom-poc-hardening/axiom.yaml" | cut -d ' ' -f 1)
"$temporary/lingo" project reopen --slug axiom-poc-hardening
"$temporary/lingo" project update --slug axiom-poc-hardening --name "Axiom POC hardening verified"
"$temporary/lingo" project validate --slug axiom-poc-hardening

record_count=$(find "$LINGO_STATE_ROOT/projects" -name installation.json -type f | wc -l | tr -d ' ')
if [[ "$record_count" != 1 ]]; then
  exit 1
fi
record_path=$(find "$LINGO_STATE_ROOT/projects" -name installation.json -type f -print)
record_sha=$(shasum -a 256 "$record_path" | cut -d ' ' -f 1)
portable_after_update_sha=$(shasum -a 256 "$LINGO_PROJECTS_ROOT/axiom-poc-hardening/axiom.yaml" | cut -d ' ' -f 1)
if "$temporary/lingo" project reopen --slug axiom-poc-hardening; then
  exit 1
fi
printf '{"evidence":"poc_dogfood","portable_before_install_sha256":"%s","portable_after_install_sha256":"%s","portable_after_update_sha256":"%s","local_record_sha256":"%s","result":"pass"}\n' \
  "$portable_before_sha" "$portable_after_install_sha" "$portable_after_update_sha" "$record_sha"
