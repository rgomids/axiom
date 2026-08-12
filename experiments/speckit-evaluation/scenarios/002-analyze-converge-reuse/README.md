# Scenario 002 — Analyze/Converge Capability Reuse

> TEMPORARY EXPERIMENT — not Axiom product behavior or architecture.

## Question

Which approach better satisfies Axiom's need to detect inconsistencies across
specification, plan, tasks, implementation, and decisions?

- B — encapsulate pinned Spec-Kit analyze/converge behind a translation boundary;
- C — implement an Axiom-native, conceptually compatible capability.

The current Axiom workflow is measured first as a baseline. Only afterward may
experimental capabilities be added under this scenario directory.

## Controlled faults

The frozen fixture contains exactly five intentional semantic faults:

1. rollback requirement and plan coverage without a task, implementation, or test;
2. Redis cache task/implementation without a requirement;
3. generated identifier behavior contradicting a user-supplied identifier requirement;
4. an acceptance criterion with no measurable threshold;
5. PostgreSQL introduced without an architectural decision.

The private scoring oracle is excluded from executor workspaces.

## Boundaries

- No Lingo, provider, database, service, Axiom runtime, or production architecture.
- No permanent dependency and no official `.specify/` state.
- All prototypes, fixtures, outputs, and scripts stay in this directory.
- ADR-0002 remains Proposed and requires human review.
- Scenario inputs are immutable after `scenario/checksums.sha256` is created.

## Layout

- `scenario/`: frozen neutral artifacts shown to each approach.
- `protocol/`: neutral contract, schema, pre-run decision rules, and prompts.
- `oracle/`: expected findings used only for scoring.
- `capabilities/`: experimental Axiom-native capability, added after baseline.
- `adapters/`: confined Spec-Kit translation prototype, added after baseline.
- `evidence/`: raw runs, normalized results, metrics, failures, and comparison.
- `scripts/`: deterministic validation and scoring utilities.

## Result

- [Seeded detection](evidence/accuracy.md): C experimental 4/5 expected with one
  unexpected; B 3/5 with three unexpected. Unexpected findings are classified
  separately from validated false positives.
- [Unexpected findings review](evidence/unexpected-findings-review.md): all
  eight current unexpected findings are out-of-scope observations; none is a
  validated false positive.
- [Metrics](evidence/metrics.md): C used 77,514 input tokens and 89.82s; B used
  193,169 and 179.13s including preparation.
- [Adapter failures](evidence/adapter-failures.md): unavailable, incompatible,
  capability-error, mutation, cleanup, and upgrade behavior.
- [Multi-repository analysis](evidence/multi-repository.md): trade-offs for one
  Project spanning backend, frontend, and infra.
- [Scorecard](evidence/scorecard.md) and [result](comparison/result.md): C remains
  the leading hypothesis; ADR-0002 remains Proposed.
