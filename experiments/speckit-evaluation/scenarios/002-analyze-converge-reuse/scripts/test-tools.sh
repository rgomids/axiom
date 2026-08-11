#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
scenario_root=$(CDPATH='' cd -- "$script_dir/.." && pwd)
test_dir=$(mktemp -d /tmp/axiom-scenario002-tools.XXXXXX)
trap 'rm -rf "$test_dir"' EXIT

"$script_dir/validate-findings.sh" "$scenario_root/oracle/expected-findings.json" >/dev/null

jq '.interactions = 0' "$scenario_root/oracle/expected-findings.json" \
  > "$test_dir/invalid.json"
if "$script_dir/validate-findings.sh" "$test_dir/invalid.json" >/dev/null 2>&1; then
  echo "expected invalid result rejection" >&2
  exit 1
fi

jq '{
  approach:"test",
  artifacts_consulted:["scenario/specification.md"],
  interactions:1,
  human_decisions_needed:[],
  limitations:[],
  findings:([.findings[0]] + [{
    id:"TST-999",
    severity:"minor",
    category:"task_coverage",
    artifact:"scenario/tasks.md",
    evidence:"controlled unexpected finding",
    subject_reference:"T-999",
    requirement_reference:null,
    recommendation:"none",
    confidence:1,
    human_decision_required:false
  }])
}' "$scenario_root/oracle/expected-findings.json" > "$test_dir/actual.json"

"$script_dir/score-findings.sh" \
  "$scenario_root/oracle/expected-findings.json" "$test_dir/actual.json" \
  > "$test_dir/score.json"

jq -e '
  .expected_findings == 5 and
  .detected_findings == 1 and
  .missed_findings == 4 and
  .unexpected_findings == 1
' "$test_dir/score.json" >/dev/null

echo "tool_tests=passed"
