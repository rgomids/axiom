# Architecture Decision Records

Este diretório registra decisões técnicas duráveis e difíceis de reverter.

## Index

- [ADR-0001 — Project is not Repository](0001-project-is-not-repository.md) — Accepted, 2026-08-08.
- [ADR-0002 — Axiom Relationship with GitHub Spec-Kit](0002-axiom-speckit-relationship.md) — Accepted, 2026-09-10.
- [ADR-0003 — Lingo as Axiom Local Control Plane](0003-lingo-as-axiom-local-control-plane.md) — Accepted, 2026-09-10; implementation requires an approved Specification.
- [ADR-0004 — Portable Project Manifest](0004-portable-project-manifest.md) — Accepted, 2026-09-12; versioned portable intent, identity/evolution and local-state separation, with explicit Git authority boundaries.
- [ADR-0005 — Bounded local filesystem threat model](0005-bounded-local-filesystem-threat-model.md) — Accepted, 2026-09-20; exact supported local filesystem properties and explicit same-UID/power-loss/media exclusions from HD-3.
- [ADR-0006 — Machine-local detail artifacts](0006-machine-local-detail-artifacts.md) — Accepted, 2026-09-20; central local ownership, Execution-first correlation, Evidence references, retention and safe cleanup from HD-2.

## Candidate assessment

| Candidate | Current classification | ADR now? |
|---|---|---|
| Codex-first harness | Accepted bounded product scope for current harness; not a permanent single-runtime commitment. | No. Existing scope is explicit and reversible. |
| Go for future CLI | Go foundations implemented; CLI framework and distribution remain undecided. | Later, with an approved CLI specification and alternatives evidence. |
| Provider abstraction | Accepted boundary principle; concrete ports and adapters remain open. | Later, when a specified integration creates a durable contract. |
| Local versus remote control-plane topology | Local Lingo direction accepted; remote and hybrid alternatives remain revisit paths. | Accepted in ADR-0003; detailed behavior still requires Specification evidence. |
| Portable Project Manifest | Versioned shareable Project intent, currently axiom.yaml, distinct from local state. | Accepted in ADR-0004; concrete v1 contract belongs to Specification 002/Plan. Identity/location, incremental update and optional Git backing authority extend this boundary; concrete storage/sync engines and aggregate ownership remain open. |
| Spec-Kit relationship | Independent Axiom implementation informed by Spec-Kit as a strategic upstream reference. | Accepted in ADR-0002; future adapters or compatibility contracts require separate evidence and approval. |
| Execution and Evidence model | Broad model remains open. ADR-0006 resolves only machine-local detail-artifact ownership, Execution-first/pre-Execution correlation, retention and cleanup. | No new broad ADR until a specified workflow requires the remaining schema/privacy choices. |
| Local filesystem threat model | Bounded local MVP model with required confinement, process concurrency, deterministic faults, complete canonical state and guided recovery; arbitrary malicious same-UID interleavings and physical/media durability excluded. | Accepted in ADR-0005; concrete mechanisms remain Plan/implementation work. |
| Lingo local control plane | Accepted architectural direction separating executable workflows from the Axiom domain and thin runtime skills. | Accepted in ADR-0003; do not implement before an approved Specification. |

ADRs futuros devem registrar, no mínimo:

- status;
- contexto;
- decisão;
- alternativas consideradas;
- consequências e trade-offs;
- evidências ou specifications relacionadas.

Hipóteses de pesquisa não são decisões. Não crie um ADR apenas para preencher a árvore documental.
