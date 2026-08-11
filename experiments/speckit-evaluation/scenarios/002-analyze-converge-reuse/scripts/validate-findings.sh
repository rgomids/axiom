#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 RESULT_JSON" >&2
  exit 64
fi

jq -e '
  type == "object" and
  (.approach | type == "string" and length > 0) and
  (.artifacts_consulted | type == "array" and all(.[]; type == "string" and length > 0)) and
  ((.artifacts_consulted | unique | length) == (.artifacts_consulted | length)) and
  (.interactions | type == "number" and . >= 1 and floor == .) and
  (.human_decisions_needed | type == "array" and all(.[]; type == "string" and length > 0)) and
  (.limitations | type == "array" and all(.[]; type == "string" and length > 0)) and
  (.findings | type == "array") and
  ([.findings[].id] | length == (unique | length)) and
  all(.findings[];
    (.id | test("^[A-Z]+-[0-9]{3}$")) and
    (.severity | IN("blocker", "critical", "major", "minor", "note")) and
    (.category | IN(
      "requirement_coverage",
      "task_coverage",
      "contradiction",
      "unsupported_behavior",
      "missing_decision",
      "unverifiable_acceptance"
    )) and
    (.artifact | type == "string" and length > 0) and
    (.evidence | type == "string" and length > 0) and
    (.subject_reference | type == "string" and length > 0) and
    ((.requirement_reference == null) or
      (.requirement_reference | type == "string" and length > 0)) and
    (.recommendation | type == "string" and length > 0) and
    (.confidence | type == "number" and . >= 0 and . <= 1) and
    (.human_decision_required | type == "boolean")
  )
' "$1" >/dev/null

echo "findings_status=valid"
