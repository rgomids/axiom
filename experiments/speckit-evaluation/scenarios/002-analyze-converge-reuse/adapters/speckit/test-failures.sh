#!/usr/bin/env bash
set -euo pipefail

adapter_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
scenario_root=$(CDPATH='' cd -- "$adapter_dir/../.." && pwd)
test_dir=$(mktemp -d /tmp/axiom-scenario002-adapter.XXXXXX)
trap 'rm -rf "$test_dir"' EXIT

set +e
SPECIFY_LAUNCHER=/path/that/does/not/exist \
  "$adapter_dir/prepare-workspace.sh" \
  "$scenario_root" "$test_dir/unavailable" v0.16.2 \
  > "$test_dir/unavailable.out" 2> "$test_dir/unavailable.err"
unavailable_exit=$?
set -e
[[ $unavailable_exit -eq 69 ]]
grep -q 'adapter_error=spec_kit_unavailable' "$test_dir/unavailable.err"

set +e
"$adapter_dir/normalize-result.sh" \
  "$scenario_root/scripts/validate-findings.sh" \
  "$adapter_dir/test-fixtures/incompatible-output.json" \
  "$test_dir/must-not-exist.json" \
  > "$test_dir/incompatible.out" 2> "$test_dir/incompatible.err"
incompatible_exit=$?
set -e
[[ $incompatible_exit -eq 65 ]]
[[ ! -e "$test_dir/must-not-exist.json" ]]
grep -q 'adapter_error=incompatible_output' "$test_dir/incompatible.err"

cp "$scenario_root/oracle/expected-findings.json" "$test_dir/existing.json"
existing_hash=$(shasum -a 256 "$test_dir/existing.json" | awk '{print $1}')
set +e
"$adapter_dir/normalize-result.sh" \
  "$scenario_root/scripts/validate-findings.sh" \
  "$scenario_root/oracle/expected-findings.json" \
  "$test_dir/existing.json" \
  > "$test_dir/existing.out" 2> "$test_dir/existing.err"
existing_exit=$?
set -e
[[ $existing_exit -eq 73 ]]
[[ $(shasum -a 256 "$test_dir/existing.json" | awk '{print $1}') == "$existing_hash" ]]
grep -q 'adapter_error=output_exists' "$test_dir/existing.err"

mkdir -p "$test_dir/workspace/.experiment/protocol"
cp "$scenario_root/protocol/findings.schema.json" \
  "$test_dir/workspace/.experiment/protocol/"
cp "$scenario_root/protocol/speckit-encapsulated-prompt.md" \
  "$test_dir/workspace/.experiment/protocol/"

set +e
CODEX_EXECUTABLE="$adapter_dir/test-fixtures/failing-codex.sh" \
  "$adapter_dir/run-capability.sh" \
  "$test_dir/workspace" "$test_dir/events.jsonl" "$test_dir/result.json" \
  "$test_dir/time.txt" gpt-5.6-sol \
  > "$test_dir/failure.out" 2> "$test_dir/failure.err"
capability_exit=$?
set -e
[[ $capability_exit -eq 70 ]]
[[ ! -e "$test_dir/result.json" ]]
grep -q 'adapter_error=capability_failure' "$test_dir/failure.err"

echo "adapter_failure_tests=passed"
