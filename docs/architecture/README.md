# Architecture

Axiom define produto, domínio, políticas e contratos. Lingo é a direção aceita
para executá-los como control plane local, conforme a ADR-0003. A ADR-0004 define
o Portable Project Manifest separado do estado local. Direção arquitetural aceita
e fundação implementada não equivalem a uma aplicação operacional.

## Target architecture

A [Specification 002](../specifications/002-lingo-project-initialization/spec.md)
e seu [Plan](../specifications/002-lingo-project-initialization/plan.md#1-architecture-and-minimal-organization)
descrevem CLI → aplicação → domínio/contratos → ports → adapters. Dependências de
código apontam para dentro: domínio não depende de codecs, filesystem ou vendors;
a aplicação define os ports que consome. Apresentação e adapters concretos de
filesystem ainda precisam de implementação autorizada e Evidence.

Contexto de orientação para agentes: [../../.agents/context/axiom-architecture.md](../../.agents/context/axiom-architecture.md).
Specifications e ADRs aprovados continuam sendo fontes canônicas.

## Implemented foundation

O módulo Go contém quatro pacotes com testes:

| Pacote | Responsabilidade implementada |
|---|---|
| [`internal/project`](../../internal/project/) | Domínio e invariantes de Project; identidade imutável, declarações e materialização/validação de estado completo |
| [`internal/projectapp`](../../internal/projectapp/) | Contratos de aplicação, ports, snapshots/revisões, authority vinculada ao preview e resultados de mutação |
| [`internal/manifest`](../../internal/manifest/) | Codec estrito do manifesto portátil `axiom.yaml`, com mapeamento para domínio e serialização canônica em memória |
| [`internal/local`](../../internal/local/) | Codec estrito do registro local JSON, metadados e distinção entre ausência e registro inválido, em memória |

`internal/projectapp` não entrega casos de uso completos. `internal/local` ainda
não implementa armazenamento em filesystem. Testes de codecs e contratos com
fakes não provam persistência, CAS físico, confinamento, symlink/TOCTOU ou recovery.
O [índice da Specification](../specifications/README.md#002--lingo-project-initialization)
aponta para Tasks, Evidence e estado de aceitação.

H12/Option B preserva `LocalRevision` calculado sobre bytes observados, nunca
persistido no registro; `portableRevision` identifica independentemente o snapshot
portátil validado. Detalhes pertencem às
[clarifications](../specifications/002-lingo-project-initialization/clarifications.md#single-local-revision-decision--2026-09-16).

## Not implemented

Não existem aplicação/CLI Lingo executável, persistência em filesystem,
instalação operacional, runtime/provider adapters ou orchestration. Framework CLI,
engine de persistência, runtime e provider não são escolhidos por este panorama.

## Architecture references

- [Conceptual Model](conceptual-model.md) — linguagem de domínio, relações, limites e questões abertas.
- [Provider Boundaries](provider-boundaries.md) — limites de capabilities e abstrações prematuras a evitar.
- [ADR-0001 — Project is not Repository](../decisions/0001-project-is-not-repository.md) — Accepted; hierarquia e ownership globais permanecem abertos.
- [ADR-0002 — Axiom Relationship with GitHub Spec-Kit](../decisions/0002-axiom-speckit-relationship.md) — Accepted; Axiom independente, Spec-Kit como referência estratégica de pesquisa.
- [ADR-0003 — Lingo as Axiom Local Control Plane](../decisions/0003-lingo-as-axiom-local-control-plane.md) — Accepted; separação entre domínio Axiom e execução local Lingo.
- [ADR-0004 — Portable Project Manifest](../decisions/0004-portable-project-manifest.md) — Accepted; intenção portátil versionada e estado local separados.

Decisões duráveis registram contexto e trade-offs em ADRs. Diagramas versionados
usam C4 como orientação e distinguem alvo, implementação existente e propostas.
