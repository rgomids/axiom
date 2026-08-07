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

## ADR template

```markdown
# ADR-NNN — Title

## Status
Proposed | Accepted | Superseded | Rejected

## Context

## Decision

## Alternatives considered

## Consequences

### Positive

### Negative / trade-offs

## Revisit when
```

Do not fake consensus. A proposal remains proposed until accepted.
