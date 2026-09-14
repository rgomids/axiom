# Architecture

O Axiom ainda não possui arquitetura de aplicação implementada. A separação entre o domínio Axiom e o Control Plane local Lingo é uma direção **Accepted** na ADR-0003 desde 2026-09-10. Essa aceitação não autoriza implementação; Specification e Plan precisam dos respectivos gates humanos.

Direções conceituais atuais ficam em [../../.agents/context/axiom-architecture.md](../../.agents/context/axiom-architecture.md). Elas orientam investigação e desenho, mas não substituem specifications ou ADRs aprovados.

## Current foundation

- [Conceptual Model](conceptual-model.md) — domain language, relationships, boundaries, lifecycles and open questions.
- [Provider Boundaries](provider-boundaries.md) — justified capability boundaries and premature abstractions to avoid.
- [ADR-0001 — Project is not Repository](../decisions/0001-project-is-not-repository.md) — accepted distinction; aggregate and ownership remain open.
- [ADR-0002 — Axiom Relationship with GitHub Spec-Kit](../decisions/0002-axiom-speckit-relationship.md) — accepted independent Axiom implementation informed by a strategic upstream reference.
- [ADR-0003 — Lingo as Axiom Local Control Plane](../decisions/0003-lingo-as-axiom-local-control-plane.md) — Accepted, 2026-09-10; implementation remains subject to Specification and Plan approval.

Quando uma decisão arquitetural durável existir:

1. registre contexto e trade-offs em `docs/decisions/`;
2. mantenha diagramas Mermaid ou C4 junto da documentação relevante;
3. distinga claramente estado proposto, aceito, substituído ou rejeitado;
4. preserve explicitamente a diferença entre direção proposta e arquitetura aceita.
