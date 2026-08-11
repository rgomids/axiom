# Accuracy Evidence

The frozen oracle contains five expected findings. A detection requires the
same neutral category and an actual `subject_reference` containing the expected
stable reference. Everything else is an unexpected finding. In this controlled
fixture, unexpected findings are counted as false positives even when they
could be useful observations in a real executable repository.

## Summary

| Run | Expected | Detected | Missed / false negatives | Unexpected / false positives |
|---|---:|---:|---:|---:|
| Current Axiom baseline | 5 | 4 | 1 | 4 |
| C — experimental Axiom-native | 5 | 4 | 1 | 1 |
| B — encapsulated Spec-Kit | 5 | 3 | 2 | 3 |

All matched expected findings retained `major` neutral severity.

## Finding-level result

| Expected finding | Current baseline | C — native | B — encapsulated |
|---|---|---|---|
| EXP-001: rollback coverage, `FR-004` | AXC-006 | AXN-003 | SPK-003 |
| EXP-002: Redis without requirement, `T-003` | Missed | Missed | Missed |
| EXP-003: identifier contradiction, `FR-001` | AXC-001 | AXN-001 | SPK-001 |
| EXP-004: unverifiable performance, `AC-003` | AXC-008 | AXN-005 | SPK-002 |
| EXP-005: PostgreSQL without decision, `PLAN-002` | AXC-007 | AXN-004 | Missed by explicit translation loss |

The Redis behavior was mentioned by both Axiom runs only as implementation or
decision context. Neither emitted the required root cause: the task and
implementation have no product requirement. B did not report it at all.

## Unexpected findings

- Current baseline: HTTP adapter, concrete PostgreSQL adapter/conflict mapping,
  concrete Redis adapter, and bodyless tests.
- C native: duplicate-key conflict mapping/bodyless conflict test.
- B: duplicate-key conflict mapping, HTTP adapter, and bodyless tests.

These findings arise from treating the deliberately non-executable semantic
fixture as incomplete production code. C's experimental rule to respect the
fixture boundary reduced this noise but did not eliminate it.

Machine-readable scores and normalized results are preserved under
`evidence/runs/<approach>/`.
