#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
  echo "usage: $0 EXPECTED_JSON ACTUAL_JSON [UNEXPECTED_REVIEW_JSON]" >&2
  exit 64
fi

expected=$1
actual=$2
review=${3:-/dev/null}
review_provided=false
script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)

if [[ $# -eq 3 ]]; then
  review_provided=true
  "$script_dir/validate-unexpected-review.sh" "$review" >/dev/null
fi

jq -n \
  --argjson review_provided "$review_provided" \
  --slurpfile expected "$expected" \
  --slurpfile actual "$actual" \
  --slurpfile review "$review" '
  def references:
    gsub("[;/]"; "\n") |
    split("\n") |
    map(gsub("^[[:space:]]+|[[:space:]]+$"; ""));
  def matches($expected_finding; $actual_finding):
    $actual_finding.category == $expected_finding.category and
    (($actual_finding.subject_reference | references) |
      index($expected_finding.subject_reference) != null);
  ($expected[0].findings) as $e |
  ($actual[0].findings) as $a |
  (($review[0].reviews // []) |
    map(select(.approach == $actual[0].approach))) as $approach_review |
  ($e | map(. as $expected_finding |
    ([ $a[] | select(matches($expected_finding; .)) ] | first // null) as $match |
    select($match != null) |
    . + {actual_finding_id: $match.id}
  )) as $detected |
  ($e | map(. as $expected_finding | select(any($a[];
    matches($expected_finding; .)
  ) | not))) as $missed |
  ($a | map(. as $actual_finding | select(any($e[];
    matches(.; $actual_finding)
  ) | not))) as $unexpected |
  if $review_provided and
    (($approach_review | length) != ($unexpected | length) or
     any($approach_review[];
       . as $classification |
       any($unexpected[];
         .id == $classification.finding_id and
         .category == $classification.original_category and
         .subject_reference == $classification.subject
       ) | not))
  then error("unexpected review does not exactly cover this approach")
  else .
  end |
  ($unexpected | map(. as $finding |
    ($approach_review |
      map(select(.finding_id == $finding.id)) |
      first // {classification: "unclassified"}) as $classification |
    . + {classification: $classification.classification}
  )) as $classified_unexpected |
  ($classified_unexpected |
    map(select(.classification == "valid_additional_finding")) |
    length) as $valid_additional |
  ($classified_unexpected |
    map(select(.classification == "false_positive")) |
    length) as $false_positives |
  ($classified_unexpected |
    map(select(.classification == "out_of_scope_observation")) |
    length) as $out_of_scope |
  ($classified_unexpected |
    map(select(.classification == "duplicate")) |
    length) as $duplicates |
  ($classified_unexpected |
    map(select(.classification == "unclassified")) |
    length) as $unclassified |
  {
    expected_findings: ($e | length),
    detected_findings: ($detected | length),
    missed_findings: ($missed | length),
    unexpected_findings: ($classified_unexpected | length),
    false_negatives: ($missed | length),
    false_positives: $false_positives,
    seeded_recall_ratio: (if ($e | length) == 0 then 1 else (($detected | length) / ($e | length)) end),
    unexpected_classification: {
      valid_additional_findings: $valid_additional,
      false_positives: $false_positives,
      out_of_scope_observations: $out_of_scope,
      duplicates: $duplicates,
      unclassified: $unclassified
    },
    detected: ($detected | map({id, category, subject_reference, severity, actual_finding_id})),
    missed: ($missed | map({id, category, subject_reference, severity})),
    unexpected: ($classified_unexpected |
      map({id, category, subject_reference, severity, classification}))
  }
'
