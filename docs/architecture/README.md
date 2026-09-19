# Architecture

Axiom define produto, domínio, políticas e contratos. Lingo executa um POC local
limitado, conforme a ADR-0003. A ADR-0004 define o Portable Project Manifest
separado do estado local. O POC não implementa toda a Specification 002.

## Target architecture

A [Specification 002](../specifications/002-lingo-project-initialization/spec.md)
e seu [Plan](../specifications/002-lingo-project-initialization/plan.md#1-architecture-and-minimal-organization)
descrevem CLI → aplicação → domínio/contratos → ports → adapters. Dependências de
código apontam para dentro: domínio não depende de codecs, filesystem ou vendors;
a aplicação define os ports que consome. Apresentação e adapters concretos do
POC existem; a cobertura completa de filesystem e Evidence ainda precisa
cumprir o Plan.

Contexto de orientação para agentes: [../../.agents/context/axiom-architecture.md](../../.agents/context/axiom-architecture.md).
Specifications e ADRs aprovados continuam sendo fontes canônicas.

## Implemented foundation

O módulo Go contém estes pacotes com testes:

| Pacote | Responsabilidade implementada |
|---|---|
| [`internal/project`](../../internal/project/) | Domínio e invariantes de Project; identidade imutável, declarações e materialização/validação de estado completo |
| [`internal/projectapp`](../../internal/projectapp/) | Contratos de aplicação, ports, snapshots/revisões, authority vinculada ao preview e resultados de mutação |
| [`internal/manifest`](../../internal/manifest/) | Codec estrito do manifesto portátil `axiom.yaml`, com mapeamento para domínio e serialização canônica em memória |
| [`internal/local`](../../internal/local/) | Codecs/stores locais estritos para instalação, bindings, Work Items e workflow; operações ancoradas e estado local separado |
| [`internal/codexruntime`](../../internal/codexruntime/) | Instalação/inspeção das cinco skills globais thin-entrypoint do Codex |
| [`internal/workitem`](../../internal/workitem/) | Casos de uso provider-neutral e adapter GitHub limitado para create/select/show/comment/close |
| [`internal/workflow`](../../internal/workflow/) | Gates sequenciais persistentes, Evidence por digest, interrupção/retomada e completion autorizada |
| [`internal/cli`](../../internal/cli/) e [`cmd/lingo`](../../cmd/lingo/) | Superfícies humana/JSON para Project, Runtime, Work Item e workflow |

`internal/projectapp` entrega apenas o lifecycle mínimo. `internal/local` usa
`os.Root` e `golang.org/x/sys/unix@v0.44.0` para operações de filesystem em
Linux/macOS. Testes de codecs e contratos com fakes não provam persistência;
os testes de filesystem do POC cobrem somente os casos registrados em
[Evidence](../specifications/002-lingo-project-initialization/evidence-poc.md).
O [índice da Specification](../specifications/README.md#002--lingo-project-initialization)
aponta para Tasks, Evidence e estado de aceitação.

H12/Option B preserva `LocalRevision` calculado sobre bytes observados, nunca
persistido no registro; `portableRevision` identifica independentemente o snapshot
portátil validado. Detalhes pertencem às
[clarifications](../specifications/002-lingo-project-initialization/clarifications.md#single-local-revision-decision--2026-09-16).

## Not implemented

O POC não implementa outros Runtimes/Providers, workflow multi-agent/paralelo,
package manager/auto-update, sincronização remota, documentos opcionais/rename
completos da Specification 002, nem recuperação automática. Garantias completas
contra power loss, todas as falhas de filesystem e races hostis same-UID seguem
abertas em #19/#20. Nenhum engine geral de persistência foi adotado.

## Architecture references

- [Conceptual Model](conceptual-model.md) — linguagem de domínio, relações, limites e questões abertas.
- [Provider Boundaries](provider-boundaries.md) — limites de capabilities e abstrações prematuras a evitar.
- [ADR-0001 — Project is not Repository](../decisions/0001-project-is-not-repository.md) — Accepted; hierarquia e ownership globais permanecem abertos.
- [ADR-0002 — Axiom Relationship with GitHub Spec-Kit](../decisions/0002-axiom-speckit-relationship.md) — Accepted; Axiom independente, Spec-Kit como referência estratégica de pesquisa.
- [ADR-0003 — Lingo as Axiom Local Control Plane](../decisions/0003-lingo-as-axiom-local-control-plane.md) — Accepted; separação entre domínio Axiom e execução local Lingo.
- [ADR-0004 — Portable Project Manifest](../decisions/0004-portable-project-manifest.md) — Accepted; intenção portátil versionada e estado local separados.

Decisões duráveis registram contexto e trade-offs em ADRs. Diagramas versionados
usam C4 como orientação e distinguem alvo, implementação existente e propostas.
