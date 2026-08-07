# Axiom

Axiom é uma plataforma para governar o ciclo de desenvolvimento de software assistido por IA. Seu objetivo é preservar coerência entre intenção de produto, specifications, arquitetura, ADRs, tarefas, múltiplos repositórios, implementação, validação, documentação, release e evidências operacionais.

O repositório começa com um harness Codex para exercitar esses fluxos antes de automatizá-los. A CLI do Axiom está planejada para Go, mas ainda não existe: este bootstrap não inicializa aplicação, `go.mod`, framework, banco ou infraestrutura.

## Current state

- harness Codex operacional em `.agents/`;
- contexto de produto, arquitetura e pesquisa separado de decisões aprovadas;
- documentação durável organizada em `docs/`;
- validação local do pacote e de arquivos potencialmente sensíveis;
- nenhum código de aplicação ou runtime implantável.

## Documentation

Product discovery and the current project definition are maintained in Notion during the initial discovery phase:
https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979
Durable technical decisions, specifications and architecture artifacts should progressively be versioned in this repository as the project evolves.

Comece por [docs/product/README.md](docs/product/README.md). Regras operacionais de segurança ficam em [docs/security/repository-security.md](docs/security/repository-security.md), e o bootstrap local em [docs/development/getting-started.md](docs/development/getting-started.md).

## Agent workflow

Antes de alterar o projeto, leia [AGENTS.md](AGENTS.md) e carregue apenas as skills relevantes em `.agents/skills/`. O harness é Codex-first; conceitos de domínio e blueprints devem permanecer vendor-neutral quando prático.

## Validation

No diretório raiz:

```bash
./scripts/validate-repository.sh .
./scripts/validate-agent-package.sh .
./scripts/test-validate-agent-package.sh
./scripts/test-check-sensitive-files.sh
./scripts/check-sensitive-files.sh .
```

Comandos adicionais estão em [docs/commands.md](docs/commands.md).

## Repository structure

```text
.agents/              Codex context, policies, skills, and templates
docs/                 Durable project documentation
scripts/              Deterministic repository checks
AGENTS.md              Agent entrypoint and policy router
SECURITY.md            Vulnerability reporting policy
CHANGELOG.md           Relevant repository changes
```

## Stack

O estado inicial usa Markdown, Bash, Git, GitHub e um harness Codex. Go é apenas a linguagem planejada para uma futura CLI; não é uma dependência deste bootstrap.
