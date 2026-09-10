# Axiom and GitHub Spec-Kit Evaluation

> TEMPORARY EXPERIMENT
> This directory exists to preserve the evidence used to evaluate
> Axiom's relationship with GitHub Spec-Kit.
> It is intentionally committed for review and reproducibility.
> After the resulting architectural decision is accepted and the
> durable findings are moved to docs/research and/or an ADR, this
> directory may be deleted in a separate change.

This directory contains controlled comparisons of Axiom's current workflow
and the official GitHub Spec-Kit workflow. It is evidence for a proposed
architectural decision, not an Axiom dependency or product implementation.

## Boundaries

- No Lingo or Axiom application code is implemented.
- Spec-Kit is evaluated in an isolated experimental worktree and is not added
  as an Axiom dependency.
- Both executions use the same frozen scenario and completion criteria.
- Durable conclusions live under `docs/research/`; this directory preserves
  operational evidence and may later be removed as one unit.

## Scenarios

- [Scenario 001 — Full SDD comparison](scenarios/001-full-sdd-comparison/README.md)
  indexes the original frozen experiment. Its existing `scenario/`, `axiom/`,
  `speckit/`, `comparison/`, and `evidence/` paths remain physically untouched
  so review evidence and links are not rewritten.
- [Scenario 002 — Analyze/converge capability reuse](scenarios/002-analyze-converge-reuse/README.md)
  compares a current-Axiom baseline, an experimental Axiom-native capability,
  and a confined Spec-Kit adapter against controlled semantic inconsistencies.

## Layout

- `scenario/`, `axiom/`, `speckit/`, `comparison/`, and `evidence/`: physically
  preserved Scenario 001 inputs and evidence.
- `scenarios/`: scenario index plus isolated Scenario 002 inputs, prototypes,
  executions, normalized results, and evidence.

## Reproduction status

Protocol and reproduction commands are recorded in `evidence/commands.md`.
Upstream Spec-Kit version and commit are pinned in `evidence/versions.md`.

## Current result

- Evaluated Spec-Kit: `v0.16.2`, commit
  `4871b485f97c7fa452ec58eba325d87536c55c34`.
- Both approaches completed the frozen scenario with one human clarification
  batch and no application code or persistent dependency.
- [Scenario 001 comparison](comparison/report.md): C (conceptual compatibility)
  ranked first after the full workflow comparison.
- [Scenario 002 comparison](scenarios/002-analyze-converge-reuse/comparison/result.md):
  C remains first after direct analyze/converge reuse testing; B detected fewer
  expected findings with higher translation, runtime, and token cost.
- [Durable research](../../docs/research/axiom-speckit-evaluation.md): survives
  later removal of this temporary directory.
- [ADR-0002](../../docs/decisions/0002-axiom-speckit-relationship.md): Proposed,
  not Accepted.

No recommendation was implemented or accepted. This directory remains intact
for PR review.
