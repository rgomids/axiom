# Axiom and GitHub Spec-Kit Evaluation

> TEMPORARY EXPERIMENT
> This directory exists to preserve the evidence used to evaluate
> Axiom's relationship with GitHub Spec-Kit.
> It is intentionally committed for review and reproducibility.
> After the resulting architectural decision is accepted and the
> durable findings are moved to docs/research and/or an ADR, this
> directory may be deleted in a separate change.

This directory contains a controlled comparison of Axiom's current workflow
and the official GitHub Spec-Kit workflow. It is evidence for a proposed
architectural decision, not an Axiom dependency or product implementation.

## Boundaries

- No Lingo or Axiom application code is implemented.
- Spec-Kit is evaluated in an isolated experimental worktree and is not added
  as an Axiom dependency.
- Both executions use the same frozen scenario and completion criteria.
- Durable conclusions live under `docs/research/`; this directory preserves
  operational evidence and may later be removed as one unit.

## Layout

- `scenario/`: frozen neutral input, seed repository, and controls.
- `axiom/`: Axiom workflow outputs and execution record.
- `speckit/`: Spec-Kit workflow outputs and execution record.
- `comparison/`: criterion-by-criterion comparison and strategy evaluation.
- `evidence/`: versions, environment, commands, observations, and inventories.

## Reproduction status

Protocol and reproduction commands are recorded in `evidence/commands.md`.
Upstream Spec-Kit version and commit are pinned in `evidence/versions.md`.

## Completed result

- Evaluated Spec-Kit: `v0.16.2`, commit
  `4871b485f97c7fa452ec58eba325d87536c55c34`.
- Both approaches completed the frozen scenario with one human clarification
  batch and no application code or persistent dependency.
- [Comparison report](comparison/report.md): C (conceptual compatibility)
  ranked first; B (encapsulation) remains a future experiment option.
- [Durable research](../../docs/research/axiom-speckit-evaluation.md): survives
  later removal of this temporary directory.
- [ADR-0002](../../docs/decisions/0002-axiom-speckit-relationship.md): Proposed,
  not Accepted.

No recommendation was implemented. This directory remains intact for PR review.
