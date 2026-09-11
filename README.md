# Axiom

Axiom é uma plataforma, domínio e conjunto de contratos para governar o ciclo de desenvolvimento de software assistido por IA. Seu objetivo é preservar coerência entre intenção de produto, specifications, arquitetura, ADRs, tarefas, múltiplos repositórios, implementação, validação, documentação, release e evidências operacionais.

O repositório começa com um harness Codex para exercitar esses fluxos antes de automatizá-los. A CLI do Axiom está planejada para Go, mas ainda não existe: este bootstrap não inicializa aplicação, `go.mod`, framework, banco ou infraestrutura.

## Current state

- harness Codex operacional em `.agents/`;
- contexto de produto, arquitetura e pesquisa separado de decisões aprovadas;
- documentação durável organizada em `docs/`;
- validação local do pacote e de arquivos potencialmente sensíveis;
- harness SDD, domínio e lifecycle próprios definidos como direção aceita, com Spec-Kit somente como referência upstream estratégica;
- Lingo aceito como Control Plane local executável na ADR-0003, sem implementação;
- Project configuration, runtime/model portability, capability negotiation e orquestração multi-agent registrados como direção para futuras Specifications;
- [Specification 002 — Lingo Project Initialization](docs/specifications/002-lingo-project-initialization/spec.md) Proposed, pronta para revisão humana final; Q1–Q6 resolvidas por revisão humana, sem Plan, Tasks ou implementação;
- nenhum código de aplicação ou runtime implantável.

## Documentation

Product discovery and the current project definition are maintained in Notion during the initial discovery phase:
https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979
Durable technical decisions, specifications and architecture artifacts are versioned in this repository as the project evolves. Notion remains a discovery source, not automatic approval.

Comece pela [fundação de produto](docs/product/foundation.md), pela [Constitution](docs/product/constitution.md) e pelo [roadmap](docs/product/roadmap.md). O [modelo conceitual](docs/architecture/conceptual-model.md), as [specifications](docs/specifications/README.md), os [ADRs](docs/decisions/README.md) e a [pesquisa](docs/research/README.md) separam estado aprovado de questões abertas. A [avaliação Axiom/Spec-Kit](docs/research/axiom-speckit-evaluation.md) preserva os cenários full-SDD e analyze/converge; a [ADR-0002](docs/decisions/0002-axiom-speckit-relationship.md) estabelece implementação Axiom independente e Spec-Kit como referência upstream estratégica. A [ADR-0003](docs/decisions/0003-lingo-as-axiom-local-control-plane.md) estabelece Lingo como Control Plane local; implementação depende de Specification aprovada. Regras operacionais de segurança ficam em [docs/security/repository-security.md](docs/security/repository-security.md), e o bootstrap local em [docs/development/getting-started.md](docs/development/getting-started.md).

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
