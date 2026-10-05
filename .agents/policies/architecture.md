# Architecture Policy

## Rules

- Preserve clear dependency direction.
- Keep domain concepts independent from vendor-specific adapters where practical.
- Prefer explicit interfaces at boundaries that have a realistic second implementation.
- Do not create interfaces solely for stylistic purity.
- Prefer cohesive modules over generic shared utilities.
- Minimize coupling between repositories.
- Avoid premature distributed architecture.
- Treat multi-repository support as a domain concern, not as a Git submodule assumption.
- Record long-lived, hard-to-reverse, cross-cutting decisions as ADRs.

## ADR trigger

Consider an ADR when the change affects one or more of:

- persistence technology;
- public contracts;
- security/trust boundary;
- runtime model;
- deployment topology;
- project/repository model;
- synchronization/source-of-truth model;
- agent execution model;
- provider abstraction;
- data ownership;
- irreversible dependency.

Do not create ADRs for routine implementation details.

## ADR evolution

Governed by [ADR-0018](../../docs/decisions/0018-adr-evolution-and-supersession-governance.md); do not add a second convention.

- Accepted ADRs are historical records: do not silently rewrite them. Spelling, formatting, and link fixes that do not change meaning are ordinary maintenance.
- A material change to an accepted decision goes in a new ADR, which stays `Proposed` until a human accepts it. Acceptance is never inferred from CI, implementation, or merge.
- Partial supersession: the earlier ADR keeps `Accepted`, the historical fragment stays visible, and the canonical annotation follows it in place. The ADR index adds `Partially superseded by`.
- Full supersession: the earlier ADR's first Status line becomes `Superseded by [ADR-NNNN — <title>](...)`, with the acceptance date.
- The superseding ADR links back to each superseded ADR in a `## Supersedes` section.
- `scripts/check-adr-governance.py` validates structure only. Whether a change semantically contradicts an accepted ADR is a review responsibility.
