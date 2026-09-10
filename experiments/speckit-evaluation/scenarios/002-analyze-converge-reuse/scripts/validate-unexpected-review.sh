#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 UNEXPECTED_REVIEW_JSON" >&2
  exit 64
fi

jq -e '
  type == "object" and
  ((keys | sort) == ["methodology", "reviews"]) and
  (.methodology | type == "string" and length > 0) and
  (.reviews | type == "array") and
  (all(.reviews[];
    type == "object" and
    ((keys | sort) == ([
      "approach",
      "evidence",
      "finding_id",
      "original_category",
      "rationale",
      "reviewer",
      "subject",
      "classification"
    ] | sort)) and
    (.approach | type == "string" and length > 0) and
    (.finding_id | type == "string" and length > 0) and
    (.original_category | type == "string" and length > 0) and
    (.subject | type == "string" and length > 0) and
    (.classification | IN(
      "valid_additional_finding",
      "false_positive",
      "out_of_scope_observation",
      "duplicate",
      "unclassified"
    )) and
    (.rationale | type == "string" and length > 0) and
    (.evidence | type == "array" and length > 0 and
      all(.[]; type == "string" and length > 0)) and
    (.reviewer | IN("human-review-required", "experiment-review"))
  )) and
  ((.reviews | map([.approach, .finding_id]) | unique | length) ==
    (.reviews | length))
' "$1" >/dev/null

echo "unexpected_review=valid"
