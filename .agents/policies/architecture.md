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
