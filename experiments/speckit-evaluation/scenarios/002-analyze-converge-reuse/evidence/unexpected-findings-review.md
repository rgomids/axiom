# Unexpected Findings Review

## Method

The five-item oracle is ground truth only for deliberately seeded faults. An
unexpected finding is therefore not automatically a false positive. Each
finding below retains its original ID, category, and subject and is reviewed
against its original text and the frozen fixture boundary.

The fixture source and tests identify themselves as non-executable experiment
artifacts. `scenario/tasks.md` also states that every listed task is represented
and no additional work is implied. Production-completeness observations can be
technically valid while remaining out of scope for this deliberately partial
semantic fixture.

The machine-readable source is
[`unexpected-findings-review.json`](unexpected-findings-review.json), validated
against
[`unexpected-findings-review.schema.json`](../protocol/unexpected-findings-review.schema.json).

## Current Axiom baseline

| Finding | Original category | Subject | Classification | Rationale |
|---|---|---|---|---|
| AXC-002 | `task_coverage` | `T-001` | `out_of_scope_observation` | HTTP routes/mappings matter for production, but executable API completeness is outside the frozen fixture. |
| AXC-003 | `task_coverage` | `T-002` | `out_of_scope_observation` | Concrete PostgreSQL and conflict behavior matter for production, but adapters and runnable completeness are outside the fixture. |
| AXC-004 | `task_coverage` | `T-003` | `out_of_scope_observation` | Concrete Redis implementation is outside scope and does not identify the seeded unsupported-behavior root cause. |
| AXC-005 | `task_coverage` | `T-004` | `out_of_scope_observation` | Executable assertions matter for production, but the tests are explicitly non-executable semantic input. |

## C — experimental Axiom-native

| Finding | Original category | Subject | Classification | Rationale |
|---|---|---|---|---|
| AXN-002 | `requirement_coverage` | `RecordService.create / T-002 / duplicate-conflict test` | `out_of_scope_observation` | Runnable conflict behavior exceeds the frozen non-executable fixture boundary. |

## B — encapsulated Spec-Kit

| Finding | Original category | Subject | Classification | Rationale |
|---|---|---|---|---|
| SPK-004 | `requirement_coverage` | `FR-002/T-002` | `out_of_scope_observation` | Conflict mapping and runnable verification exceed the frozen fixture boundary. |
| SPK-005 | `task_coverage` | `PLAN-001/T-001` | `out_of_scope_observation` | HTTP adapter completeness exceeds the frozen fixture boundary. |
| SPK-006 | `task_coverage` | `PLAN-005/T-004` | `out_of_scope_observation` | Runnable tests exceed the frozen fixture boundary. |

## Classification summary

| Run | Valid additional | False positive | Out of scope | Duplicate | Unclassified |
|---|---:|---:|---:|---:|---:|
| Current Axiom baseline | 0 | 0 | 4 | 0 | 0 |
| C — experimental Axiom-native | 0 | 0 | 1 | 0 | 0 |
| B — encapsulated Spec-Kit | 0 | 0 | 3 | 0 | 0 |

No reviewed finding asserts a nonexistent condition, so none is classified as
a validated false positive. C produced fewer unexpected and out-of-scope
observations than B; both produced zero validated false positives.
