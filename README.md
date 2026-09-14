# Axiom

Axiom é uma plataforma, domínio e conjunto de contratos para governar o ciclo de desenvolvimento de software assistido por IA. Seu objetivo é preservar coerência entre intenção de produto, specifications, arquitetura, ADRs, tarefas, múltiplos repositórios, implementação, validação, documentação, release e evidências operacionais.

O repositório começa com um harness Codex para exercitar esses fluxos antes de automatizá-los. T01 da Specification 002 entrega domínio puro em Go, com testes. A CLI ainda não existe; não há framework, banco ou infraestrutura.

## Current state

- harness Codex operacional em `.agents/`;
- contexto de produto, arquitetura e pesquisa separado de decisões aprovadas;
- documentação durável organizada em `docs/`;
- validação local do pacote e de arquivos potencialmente sensíveis;
- harness SDD, domínio e lifecycle próprios definidos como direção aceita, com Spec-Kit somente como referência upstream estratégica;
- Lingo aceito como Control Plane local executável na ADR-0003; primeiro domínio puro implementado em `internal/project/`;
- Project configuration, runtime/model portability, capability negotiation e orquestração multi-agent registrados como direção para futuras Specifications;
- [Specification 002 — Lingo Project Initialization](docs/specifications/002-lingo-project-initialization/spec.md) originalmente aprovada em 2026-09-11; reaberta/reconciliada por decisões humanas posteriores de 2026-09-12 sobre localização, id/slug/name, init mínimo, update explícito e Git backing/sync opcional. [Clarifications](docs/specifications/002-lingo-project-initialization/clarifications.md) preservam histórico Q1–Q6 e registram H1–H11: modelo do Project, init/update, atomicidade lógica e fronteira Git Authority aprovados; mecanismos de persistência e design Git permanecem futuros, sujeitos a Evidence; [Plan](docs/specifications/002-lingo-project-initialization/plan.md): **Approved**, PR #4 aprovado e mergeado em 2026-09-14. [Tasks](docs/specifications/002-lingo-project-initialization/tasks.md): **Approved** após PR #5 mergeado. **Implementation: Authorized (T01 only)**; [T01 e Evidence](docs/specifications/002-lingo-project-initialization/evidence-t01.md) prontos para human implementation review. **T02–T21: Not started**;
- [ADR-0004 — Portable Project Manifest](docs/decisions/0004-portable-project-manifest.md) Accepted: manifesto portátil/versionado do Project, working copy portátil separada do estado local por ID, evolução incremental e Git backing opcional; extensão aprovada nas decisões humanas posteriores, schema inicial detalhado no Plan;
- nenhum código de aplicação ou runtime implantável.

## Documentation

Product discovery and the current project definition are maintained in Notion during the initial discovery phase:
https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979
Durable technical decisions, specifications and architecture artifacts are versioned in this repository as the project evolves. Notion remains a discovery source, not automatic approval.

Comece pela [fundação de produto](docs/product/foundation.md), pela [Constitution](docs/product/constitution.md) e pelo [roadmap](docs/product/roadmap.md). O [modelo conceitual](docs/architecture/conceptual-model.md), as [specifications](docs/specifications/README.md), os [ADRs](docs/decisions/README.md) e a [pesquisa](docs/research/README.md) separam estado aprovado de questões abertas. A [avaliação Axiom/Spec-Kit](docs/research/axiom-speckit-evaluation.md) preserva os cenários full-SDD e analyze/converge; a [ADR-0002](docs/decisions/0002-axiom-speckit-relationship.md) estabelece implementação Axiom independente e Spec-Kit como referência upstream estratégica. A [ADR-0003](docs/decisions/0003-lingo-as-axiom-local-control-plane.md) estabelece Lingo como Control Plane local; implementação depende de Specification e Plan aprovados. Regras operacionais de segurança ficam em [docs/security/repository-security.md](docs/security/repository-security.md), e o bootstrap local em [docs/development/getting-started.md](docs/development/getting-started.md).

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

Para validar T01 com Go instalado, sem downloads de módulos/toolchain:

```bash
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go test -race ./internal/project
go vet ./...
go build ./...
go run ./scripts/check-project-domain.go
bash scripts/test-check-project-domain.sh
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
internal/project/     Pure Go Project domain and behavior tests
go.mod                Standard-library-only Go module
scripts/              Deterministic repository and domain-boundary checks
experiments/          Temporary, reviewable research evidence
AGENTS.md              Agent entrypoint and policy router
SECURITY.md            Vulnerability reporting policy
CHANGELOG.md           Relevant repository changes
```

## Stack

Stack: Go 1.26 para domínio e testes; biblioteca padrão somente. Markdown, Bash, Git, GitHub e harness Codex permanecem. Sem serviços, jobs, variáveis de ambiente de aplicação ou dependências externas.
