# Multi-repository Thought Experiment

Scenario 002 is executable in one repository. This analysis considers one
logical Project with one Specification affecting independent `backend`,
`frontend`, and `infra` repositories. It proposes no final architecture.

## B — Spec-Kit encapsulation

### One aggregate Spec-Kit project

The adapter could mirror all three repositories into one disposable feature
workspace. One analyze/converge run could see the shared specification, but the
mirror must invent a filesystem namespace, preserve repository/source SHAs,
avoid path collisions, and decide which mutated tasks belong to which real
repository. Context size and permission scope grow together. The disposable
project is not any authoritative repository.

### Three Spec-Kit projects

The adapter could project the specification into three initialized workspaces.
Each run gets natural local paths, but the shared specification is duplicated.
Cross-repository requirements can appear covered three times or nowhere, and
no run can prove end-to-end coverage. Version, template, and feature state must
remain compatible across all three workspaces.

### One run per repository plus aggregation

This reduces per-run context but requires an Axiom-owned aggregation layer.
Correlation needs at least Project ID, Repository ID, source SHA, specification
reference, requirement reference, run/version identity, and normalized finding
identity. The aggregator must deduplicate shared contradictions, distinguish a
repository-local miss from a Project-level miss, and reconcile convergence
tasks without writing them into the wrong repository.

No option removes Axiom orchestration. B either creates one synthetic Spec-Kit
project, maintains three projections, or owns aggregation around per-repository
runs. Scenario 002's lost decision artifact becomes harder when decisions span
repositories.

## C — Axiom-native

An Axiom-native capability could accept a Project-level input directly:

```text
Project specification and decisions
  + backend plan/tasks/source/evidence
  + frontend plan/tasks/source/evidence
  + infra plan/tasks/source/evidence
  -> Project finding set with repository-scoped evidence
```

It could build a global requirement matrix, delegate bounded repository scans,
then aggregate on stable Axiom references. That better matches `Project !=
Repository` and avoids translating repository identity into a foreign feature
directory.

This remains conceptual. Axiom has no executable Project registry, source
snapshot contract, cross-repository context selector, aggregator, or ownership
model. C therefore has lower model impedance, not proven implementation.

## Trade-off

B offers mature per-feature checks today but moves multi-repository complexity
into translation and aggregation. C can model the need directly but must build
the capability and deterministic correlation rules. A real executable test
becomes decision-relevant after Axiom defines minimum Project/repository input
and evidence contracts; before that, it would compare invented scaffolds.
