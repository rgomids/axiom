# Axiom Product

Axiom é uma plataforma para governar o ciclo de desenvolvimento de software assistido por IA.

Estado consolidado: [Product Foundation](foundation.md). Governança normativa: [Axiom Constitution](constitution.md). Sequenciamento futuro: [Product Roadmap](roadmap.md).

Seu objetivo é manter coerência entre:

- intenção de produto;
- especificações;
- arquitetura;
- ADRs;
- tarefas;
- múltiplos repositórios;
- implementação;
- validação;
- documentação;
- release;
- evidências operacionais.

## Project != Repository

Um projeto do Axiom poderá abranger múltiplos repositórios independentes. Um repositório é uma unidade de código e versionamento; o projeto é a unidade que conecta intenção, decisões, trabalho, evidências e entregas entre esses repositórios.

Suporte a múltiplos repositórios é um requisito de domínio. Ele não implica Git submodules nem define antecipadamente um modelo de workspace.

Decisão e trade-offs: [ADR-0001](../decisions/0001-project-is-not-repository.md). Linguagem de domínio e questões abertas: [Conceptual Model](../architecture/conceptual-model.md).

## Documentation authority

- **Notion:** visão de produto, discovery, fluxos e hipóteses em
  [Notion discovery](https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979).
  Conteúdo externo não implica aprovação ou permissão de publicação; confirme
  classificação antes de incorporá-lo ao repositório público.
- **GitHub / repositório versionado:** verdade técnica em Specifications, ADRs,
  arquitetura, código e Evidence. Artefatos aprovados governam os contratos;
  contexto de agentes resume e aponta para essas fontes.
- **GitHub Issues/Projects, quando adotados:** estado operacional do trabalho.
  Não substituem contratos aprovados nem transformam roadmap em task tracker.

Contexto resumido para agentes: [../../.agents/context/axiom-product.md](../../.agents/context/axiom-product.md).

## Current and future slices

- [Specification 002 — Lingo Project Initialization](../specifications/002-lingo-project-initialization/spec.md)
  — primeiro slice em implementação incremental: init mínimo, update explícito,
  reabertura e instalação local. Consulte o [lifecycle atual](../specifications/README.md#002--lingo-project-initialization);
  a CLI ainda não está disponível.
- [Specification 001 — Codex Agent Harness Generation](../specifications/001-codex-agent-harness-generation/spec.md)
  — Proposed, voltada ao harness.
- [Dogfooding 001 — Go Pull Request Review Agent](dogfooding/001-go-pr-review-agent.md)
  — evidência do harness.
