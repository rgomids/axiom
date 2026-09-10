# Architecture Decision Records

Este diretório registra decisões técnicas duráveis e difíceis de reverter.

## Index

- [ADR-0001 — Project is not Repository](0001-project-is-not-repository.md) — Accepted, 2026-08-08.
- [ADR-0002 — Axiom Relationship with GitHub Spec-Kit](0002-axiom-speckit-relationship.md) — Accepted, 2026-09-10.
- [ADR-0003 — Lingo as Axiom Local Control Plane](0003-lingo-as-axiom-local-control-plane.md) — Accepted, 2026-09-10; implementation requires an approved Specification.

## Candidate assessment

| Candidate | Current classification | ADR now? |
|---|---|---|
| Codex-first harness | Accepted bounded product scope for current harness; not a permanent single-runtime commitment. | No. Existing scope is explicit and reversible. |
| Go for future CLI | Proposed direction; no application implementation approved. | Later, with an approved CLI specification and alternatives evidence. |
| Provider abstraction | Accepted boundary principle; concrete ports and adapters remain open. | Later, when a specified integration creates a durable contract. |
| Local versus remote control-plane topology | Local Lingo direction accepted; remote and hybrid alternatives remain revisit paths. | Accepted in ADR-0003; detailed behavior still requires Specification evidence. |
| Project state format | Open question. | No persistence or source-of-truth model approved. |
| Spec-Kit relationship | Independent Axiom implementation informed by Spec-Kit as a strategic upstream reference. | Accepted in ADR-0002; future adapters or compatibility contracts require separate evidence and approval. |
| Execution and Evidence model | Open conceptual model. | Schema, retention and privacy evidence still missing. |
| Lingo local control plane | Accepted architectural direction separating executable workflows from the Axiom domain and thin runtime skills. | Accepted in ADR-0003; do not implement before an approved Specification. |

ADRs futuros devem registrar, no mínimo:

- status;
- contexto;
- decisão;
- alternativas consideradas;
- consequências e trade-offs;
- evidências ou specifications relacionadas.

Hipóteses de pesquisa não são decisões. Não crie um ADR apenas para preencher a árvore documental.
