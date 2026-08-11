# Neutral Analysis Capability Contract

## Need

Evaluate coherence across Axiom artifacts without assuming Spec-Kit paths,
templates, lifecycle, or terminology.

## Inputs

- specification;
- plan;
- tasks;
- implementation;
- decisions.

## Checks

- `requirement_coverage` — requirements have plan, task, implementation, and
  verification coverage when applicable;
- `task_coverage` — tasks trace to validated needs and implementation;
- `contradiction` — artifacts assert incompatible behavior;
- `unsupported_behavior` — behavior or scope lacks a requirement or decision;
- `missing_decision` — material architecture is introduced without its required
  decision evidence;
- `unverifiable_acceptance` — acceptance cannot be tested objectively.

## Output

The output MUST conform to `findings.schema.json`. Each finding contains:

- severity;
- category;
- artifact and exact evidence;
- stable subject reference;
- optional requirement reference;
- recommendation;
- confidence;
- whether human judgment is required.

The run also reports consulted artifacts, interactions, human decisions needed,
and limitations. Tool-specific raw output remains separate from normalized
findings.

## Semantics

- `blocker`: analysis cannot establish a safe interpretation or execution must stop;
- `critical`: severe correctness/security failure;
- `major`: material inconsistency that must be resolved or waived;
- `minor`: real but non-blocking quality issue;
- `note`: observation without required correction.

The contract defines Axiom's evaluation need. It does not promise artifact or
runtime compatibility with Spec-Kit.
