# Architecture

O Axiom ainda não possui arquitetura de aplicação aprovada. Este bootstrap não escolhe framework, persistência, infraestrutura, topologia de execução ou contratos da futura CLI.

Direções conceituais atuais ficam em [../../.agents/context/axiom-architecture.md](../../.agents/context/axiom-architecture.md). Elas orientam investigação e desenho, mas não substituem specifications ou ADRs aprovados.

## Current foundation

- [Conceptual Model](conceptual-model.md) — domain language, relationships, boundaries, lifecycles and open questions.
- [Provider Boundaries](provider-boundaries.md) — justified capability boundaries and premature abstractions to avoid.
- [ADR-0001 — Project is not Repository](../decisions/0001-project-is-not-repository.md) — accepted distinction; aggregate and ownership remain open.

Quando uma decisão arquitetural durável existir:

1. registre contexto e trade-offs em `docs/decisions/`;
2. mantenha diagramas Mermaid ou C4 junto da documentação relevante;
3. distinga claramente estado proposto, aceito, substituído ou rejeitado;
4. atualize esta página apenas com arquitetura realmente adotada.
