# Axiom Product

Axiom é uma plataforma para governar o ciclo de desenvolvimento de software assistido por IA.

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

## Current documentation source

Durante a fase inicial de discovery, a definição corrente de produto é mantida no Notion:

https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979

O Notion é, neste momento, referência de produto e discovery. Seu conteúdo não deve ser copiado automaticamente. Antes de versionar qualquer informação externa, verifique sua classificação e se ela pode ser pública.

Decisões técnicas duráveis, especificações e artefatos de arquitetura devem migrar progressivamente para este repositório quando forem aprovados e estiverem prontos para publicação.

Contexto resumido para agentes: [../../.agents/context/axiom-product.md](../../.agents/context/axiom-product.md).
