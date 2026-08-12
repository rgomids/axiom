# Seeded Detection Evidence

The frozen oracle contains only five deliberately seeded findings. The scorer
therefore measures **seeded recall**, not complete accuracy. A detection requires
the same neutral category and an exact stable-reference token in the actual
`subject_reference`; `/` and `;` delimit compound references. Findings outside
the oracle remain unexpected until separately classified.

`accuracy_ratio` was removed from the derived scores. `seeded_recall_ratio`
means only `detected expected / expected`; it makes no claim about ground truth
for every possible finding.

## Summary

| Run | Expected | Detected | Missed | Seeded recall | Unexpected | Valid additional | False positives | Out of scope | Duplicates |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Current Axiom baseline | 5 | 4 | 1 | 80% | 4 | 0 | 0 | 4 | 0 |
| C — experimental Axiom-native | 5 | 4 | 1 | 80% | 1 | 0 | 0 | 1 | 0 |
| B — encapsulated Spec-Kit | 5 | 3 | 2 | 60% | 3 | 0 | 0 | 3 | 0 |

No finding remains unclassified.

## Finding-level seeded result

| Expected finding | Current baseline | C — native | B — encapsulated |
|---|---|---|---|
| EXP-001: rollback coverage, `FR-004` | AXC-006 | AXN-003 | SPK-003 |
| EXP-002: Redis without requirement, `T-003` | Missed | Missed | Missed |
| EXP-003: identifier contradiction, `FR-001` | AXC-001 | AXN-001 | SPK-001 |
| EXP-004: unverifiable performance, `AC-003` | AXC-008 | AXN-005 | SPK-002 |
| EXP-005: PostgreSQL without decision, `PLAN-002` | AXC-007 | AXN-004 | Missed by explicit translation loss |

Review confirmed every scorer match is semantically correct for this fixture.
Category plus an exact stable-reference token is sufficient here because each
seeded issue has a unique category/reference pair. The scorer rejects substring
matches such as `FR-0010` for expected `FR-001` and records the matching actual
finding ID in each score. This criterion is fixture-specific, not a universal
finding-equivalence rule.

## Unexpected findings

Each unexpected finding was reviewed individually in
[`unexpected-findings-review.md`](unexpected-findings-review.md), with the
machine-readable classification in
[`unexpected-findings-review.json`](unexpected-findings-review.json).

All eight observations concern production completeness of source or tests that
the frozen scenario explicitly labels non-executable. They are classified as
out-of-scope observations, not false positives: the stated concerns may be valid
in production, but they should not count against capability quality in this
deliberately partial fixture. C produced fewer unexpected findings than B; both
produced zero validated false positives.

## Redis gap

Every approach missed EXP-002: Redis cache behavior was introduced by plan,
task, and implementation without a supporting functional requirement. Mentions
of a missing Redis adapter or missing architecture decision do not detect this
root cause.

This is research evidence for a future candidate Axiom capability: detect
introduced behavior or implementation scope that has no supporting requirement,
decision, or approved-plan rationale. Scenario 002 does not implement or
authorize that capability.

Original model results, raw events, and timing evidence remain unchanged. Only
derived scoring, classification, and interpretation were recalculated; no model
was re-executed.
