# Architecture Decision Records

Este diretório registra decisões técnicas duráveis e difíceis de reverter.

## Index

- [ADR-0001 — Project is not Repository](0001-project-is-not-repository.md) — Accepted, 2026-08-08.
- [ADR-0002 — Axiom Relationship with GitHub Spec-Kit](0002-axiom-speckit-relationship.md) — Proposed, 2026-08-11.

## Candidate assessment

| Candidate | Current classification | ADR now? |
|---|---|---|
| Codex-first harness | Accepted bounded product scope for current harness; not a permanent single-runtime commitment. | No. Existing scope is explicit and reversible. |
| Go for future CLI | Proposed direction; no application implementation approved. | Later, with an approved CLI specification and alternatives evidence. |
| Provider abstraction | Accepted boundary principle; concrete ports and adapters remain open. | Later, when a specified integration creates a durable contract. |
| Local-first versus persisted control plane | Open question / hypothesis. | No evidence for a decision. |
| Project state format | Open question. | No persistence or source-of-truth model approved. |
| Spec-Kit relationship | Comparative research complete; conceptual compatibility recommended. | ADR-0002 is Proposed and awaits human decision. |
| Execution and Evidence model | Open conceptual model. | Schema, retention and privacy evidence still missing. |

ADRs futuros devem registrar, no mínimo:

- status;
- contexto;
- decisão;
- alternativas consideradas;
- consequências e trade-offs;
- evidências ou specifications relacionadas.

Hipóteses de pesquisa não são decisões. Não crie um ADR apenas para preencher a árvore documental.
