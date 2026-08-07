---
name: axiom-govern-project
description: Orient and evolve the Axiom project while keeping product intent, decisions, repositories, tasks, and documentation coherent.
---

# Axiom Project Governance

## Use when

- starting a new Axiom work item;
- evaluating what should be built next;
- reconciling project state;
- identifying missing decisions;
- coordinating work that spans multiple repositories.

## Do not use when

- the task is a tiny implementation with already-approved scope;
- only a deterministic validation command is required.

## Procedure

1. Inspect repository status and relevant docs.
2. Identify the user outcome, not only the requested implementation.
3. Determine which project artifacts are authoritative.
4. Classify current work:
   - discovery;
   - specification;
   - architecture;
   - implementation;
   - validation;
   - documentation;
   - release.
5. Identify unresolved decisions that materially block or alter implementation.
6. Present decisions with alternatives and trade-offs.
7. Route to the smallest additional skill required.
8. Ensure resulting knowledge is persisted if it is durable.

## Output

Keep the response compact:

```text
State
Decision(s)
Next action
Evidence / unresolved risks
```

Do not generate a large roadmap unless requested.
