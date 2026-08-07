---
name: axiom-implement
description: Implement an approved Axiom task with minimal scope, executable validation, and controlled repository impact.
---

# Implement Axiom Change

## Preconditions

Before editing:

- understand acceptance criteria;
- inspect existing implementation and tests;
- identify affected repository/repositories;
- verify relevant architecture/ADR constraints;
- check working tree for unrelated user changes.

## Procedure

1. State intended files/modules.
2. Make the smallest coherent change.
3. Preserve existing patterns unless a change is intentional.
4. Add or update tests at the correct boundary.
5. Handle errors explicitly.
6. Avoid speculative abstractions.
7. Run the narrowest useful checks first.
8. Run broader project checks when feasible.
9. Inspect the final diff.
10. Route to review/documentation as needed.

## Go guidance

When Go implementation exists:

- prefer standard library where sufficient;
- use explicit interfaces at meaningful boundaries;
- keep packages cohesive;
- return errors with useful context;
- avoid global mutable state;
- prefer table tests where they improve clarity;
- keep dependency direction aligned with architecture.

Do not create `go.mod` merely because Go is the planned CLI language.
