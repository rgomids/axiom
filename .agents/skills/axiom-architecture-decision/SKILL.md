---
name: axiom-architecture-decision
description: Analyze significant Axiom architecture choices and produce an explicit trade-off decision or ADR when justified.
---

# Architecture Decision

## Use when

A choice is long-lived, cross-cutting, expensive to reverse, or changes a major boundary.

## Procedure

1. State the decision in one sentence.
2. Identify constraints.
3. List 2–4 viable alternatives.
4. Compare:
   - complexity;
   - coupling;
   - operability;
   - security;
   - cost;
   - portability;
   - reversibility;
   - migration impact.
5. Recommend one option.
6. State what would invalidate the recommendation.
7. If the threshold is met, create/update an ADR.

## Evolving an accepted ADR

Follow `docs/decisions/0018-adr-evolution-and-supersession-governance.md`. Do not rewrite an accepted ADR.

1. Write a new ADR, `Proposed` until a human accepts it.
2. Choose the scope: partial (one bounded fragment; the old ADR stays `Accepted`) or full (the whole ADR is replaced).
3. In the old ADR, keep the historical text and add the canonical annotation right after the fragment (partial), or replace the first Status line with the canonical `Superseded by` line (full), exactly as defined in ADR-0018. Apply this once the new ADR is accepted.
4. Add a `## Supersedes` section to the new ADR linking each superseded ADR.
5. Update the `docs/decisions/README.md` index with the ADR-0018 `Partially superseded by` note or `Superseded by` status.
6. Run `scripts/check-adr-governance.py`; it checks structure only, not semantic conflict.

## ADR template

```markdown
# ADR-NNNN — Title

## Status
Proposed | Accepted | Superseded | Rejected

## Context

## Decision

## Alternatives considered

## Consequences

### Positive

### Negative / trade-offs

## Revisit when

## Supersedes
<!-- Optional: only when superseding an ADR. Link each superseded ADR and name the fragment. -->
```

Do not fake consensus. A proposal remains proposed until accepted.
