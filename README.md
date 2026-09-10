# Axiom

Axiom é uma plataforma para governar o ciclo de desenvolvimento de software assistido por IA. Seu objetivo é preservar coerência entre intenção de produto, specifications, arquitetura, ADRs, tarefas, múltiplos repositórios, implementação, validação, documentação, release e evidências operacionais.

O repositório começa com um harness Codex para exercitar esses fluxos antes de automatizá-los. A CLI do Axiom está planejada para Go, mas ainda não existe: este bootstrap não inicializa aplicação, `go.mod`, framework, banco ou infraestrutura.

## Current state

- harness Codex operacional em `.agents/`;
- contexto de produto, arquitetura e pesquisa separado de decisões aprovadas;
- documentação durável organizada em `docs/`;
- validação local do pacote e de arquivos potencialmente sensíveis;
- harness SDD, domínio e lifecycle próprios definidos como direção aceita, com Spec-Kit somente como referência upstream estratégica;
- nenhum código de aplicação ou runtime implantável.

## Documentation

Product discovery and the current project definition are maintained in Notion during the initial discovery phase:
https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979
Durable technical decisions, specifications and architecture artifacts are versioned in this repository as the project evolves. Notion remains a discovery source, not automatic approval.

Comece pela [fundação de produto](docs/product/foundation.md) e pela [Constitution](docs/product/constitution.md). O [modelo conceitual](docs/architecture/conceptual-model.md), as [specifications](docs/specifications/README.md), os [ADRs](docs/decisions/README.md) e a [pesquisa](docs/research/README.md) separam estado aprovado de questões abertas. A [avaliação Axiom/Spec-Kit](docs/research/axiom-speckit-evaluation.md) preserva os cenários full-SDD e analyze/converge; a [ADR-0002](docs/decisions/0002-axiom-speckit-relationship.md) estabelece implementação Axiom independente e Spec-Kit como referência upstream estratégica, sem dependência ou garantia de compatibilidade. Regras operacionais de segurança ficam em [docs/security/repository-security.md](docs/security/repository-security.md), e o bootstrap local em [docs/development/getting-started.md](docs/development/getting-started.md).

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
  product/             Constitution, product foundation, and dogfooding
  architecture/        Conceptual boundaries and architecture
  specifications/      Desired behavior before implementation
  decisions/           Accepted or proposed durable decisions
  research/            Hypotheses and comparative evidence
scripts/              Deterministic repository checks
experiments/          Temporary, reviewable research evidence
AGENTS.md              Agent entrypoint and policy router
SECURITY.md            Vulnerability reporting policy
CHANGELOG.md           Relevant repository changes
```

## Stack

O estado inicial usa Markdown, Bash, Git, GitHub e um harness Codex. Go é apenas a linguagem planejada para uma futura CLI; não é uma dependência deste bootstrap.
