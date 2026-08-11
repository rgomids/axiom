#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 EXPECTED_JSON ACTUAL_JSON" >&2
  exit 64
fi

expected=$1
actual=$2

jq -n --slurpfile expected "$expected" --slurpfile actual "$actual" '
  ($expected[0].findings) as $e |
  ($actual[0].findings) as $a |
  ($e | map(. as $expected_finding | select(any($a[];
    .category == $expected_finding.category and
    (.subject_reference | contains($expected_finding.subject_reference))
  )))) as $detected |
  ($e | map(. as $expected_finding | select(any($a[];
    .category == $expected_finding.category and
    (.subject_reference | contains($expected_finding.subject_reference))
  ) | not))) as $missed |
  ($a | map(. as $actual_finding | select(any($e[]; . as $expected_finding |
    $expected_finding.category == $actual_finding.category and
    ($actual_finding.subject_reference | contains($expected_finding.subject_reference))
  ) | not))) as $unexpected |
  {
    expected_findings: ($e | length),
    detected_findings: ($detected | length),
    missed_findings: ($missed | length),
    unexpected_findings: ($unexpected | length),
    false_negatives: ($missed | length),
    false_positives: ($unexpected | length),
    accuracy_ratio: (if ($e | length) == 0 then 1 else (($detected | length) / ($e | length)) end),
    detected: ($detected | map({id, category, subject_reference, severity})),
    missed: ($missed | map({id, category, subject_reference, severity})),
    unexpected: ($unexpected | map({id, category, subject_reference, severity}))
  }
'
