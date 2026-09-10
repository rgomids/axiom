#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
scenario_root=$(CDPATH='' cd -- "$script_dir/.." && pwd)
test_dir=$(mktemp -d /tmp/axiom-scenario002-tools.XXXXXX)
trap 'rm -rf "$test_dir"' EXIT

"$script_dir/validate-findings.sh" "$scenario_root/oracle/expected-findings.json" >/dev/null
"$script_dir/validate-unexpected-review.sh" \
  "$scenario_root/evidence/unexpected-findings-review.json" >/dev/null

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

jq -n '{
  methodology:"test review",
  reviews:[{
    approach:"test",
    finding_id:"TST-999",
    original_category:"task_coverage",
    subject:"T-999",
    classification:"false_positive",
    rationale:"Controlled scorer classification.",
    evidence:["test fixture"],
    reviewer:"experiment-review"
  }]
}' > "$test_dir/review.json"

"$script_dir/score-findings.sh" \
  "$scenario_root/oracle/expected-findings.json" "$test_dir/actual.json" \
  "$test_dir/review.json" \
  > "$test_dir/score.json"

jq -e '
  .expected_findings == 5 and
  .detected_findings == 1 and
  .missed_findings == 4 and
  .unexpected_findings == 1 and
  .false_positives == 1 and
  .seeded_recall_ratio == 0.2 and
  .unexpected_classification == {
    valid_additional_findings:0,
    false_positives:1,
    out_of_scope_observations:0,
    duplicates:0,
    unclassified:0
  }
' "$test_dir/score.json" >/dev/null

jq '.findings[0].subject_reference = "FR-0010"' \
  "$test_dir/actual.json" > "$test_dir/substring.json"
"$script_dir/score-findings.sh" \
  "$scenario_root/oracle/expected-findings.json" "$test_dir/substring.json" \
  > "$test_dir/substring-score.json"
jq -e '
  .detected_findings == 0 and
  .unexpected_findings == 2 and
  .false_positives == 0 and
  .unexpected_classification.unclassified == 2
' "$test_dir/substring-score.json" >/dev/null

echo "tool_tests=passed"
