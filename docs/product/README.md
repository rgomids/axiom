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

## Current documentation source

Durante a fase inicial de discovery, a definição corrente de produto é mantida no Notion:

https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979

O Notion é, neste momento, referência de produto e discovery. Seu conteúdo não deve ser copiado automaticamente. Antes de versionar qualquer informação externa, verifique sua classificação e se ela pode ser pública.

Decisões técnicas duráveis, especificações e artefatos de arquitetura devem migrar progressivamente para este repositório quando forem aprovados e estiverem prontos para publicação.

Contexto resumido para agentes: [../../.agents/context/axiom-product.md](../../.agents/context/axiom-product.md).

## Current and future slices

- [Specification 001 — Codex Agent Harness Generation](../specifications/001-codex-agent-harness-generation/spec.md)
- [Dogfooding 001 — Go Pull Request Review Agent](dogfooding/001-go-pr-review-agent.md)
- [`lingo project init` candidate slice](roadmap.md#candidate-first-executable-slice) — architecture direction only; requires a Specification before implementation.
