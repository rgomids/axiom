# Product Roadmap

## Purpose

Capability direction and dependencies, not a task tracker, changelog or delivery
commitment. Approval and implementation status live in the
[Specification index](../specifications/README.md); detailed scope and Evidence
remain with each Specification. Nothing here authorizes implementation.

## Established foundation

- Axiom owns product/domain/policies/contracts; Lingo is the accepted local-first
  executable control-plane direction under [ADR-0003](../decisions/0003-lingo-as-axiom-local-control-plane.md).
- [ADR-0004](../decisions/0004-portable-project-manifest.md) establishes portable
  Project intent, immutable identity, mutable slug/name and separate local state.
- The Codex-first harness exercises the development lifecycle. Tested Go
  foundations cover Project invariants, application contracts and authority,
  portable manifests and local record codecs. See [Architecture](../architecture/README.md).

The repository now includes an accepted executable E2E Lingo/Codex POC: local
install, global thin skills, Project bindings/resolution, GitHub Work Items and a
bounded sequential workflow. Human acceptance was recorded on 2026-09-20 after
PR #52 merged to `main`. Its commands, limits and acceptance status are recorded
in the [Specification index](../specifications/README.md). Acceptance does not
authorize later roadmap work.

## MVP specification

Manual POC acceptance identified four concrete experience gaps that should inform
the MVP:

- [#54](https://github.com/rgomids/axiom/issues/54) — guided Project setup that
  asks for Project identity, repositories and Work Item Provider instead of
  inferring the caller CWD.
- [#55](https://github.com/rgomids/axiom/issues/55) — Intent-driven Work Item
  creation interview and structured issue draft.
- [#56](https://github.com/rgomids/axiom/issues/56) — provider-visible Axiom
  workflow stage plus bounded structured transition/Evidence comments.
- [#57](https://github.com/rgomids/axiom/issues/57) — explicit Runtime skill
  selectors/arguments with questions only for missing inputs.

The [approved MVP Specification](../specifications/004-mvp-v1-baseline/spec.md),
tracked by [#62](https://github.com/rgomids/axiom/issues/62), now consolidates
these experience gaps with completion output, detailed artifacts, provenance,
persistence/recovery/compatibility, installation/upgrade, and clean-environment
acceptance. Human review on 2026-09-20 recorded the four material directions:
checksummed macOS/Linux binaries with an explicit release OS/architecture matrix,
durable machine-local detail artifacts with an ADR-required ownership boundary,
the bounded filesystem threat model with mandatory Specification 002/security/
architecture reconciliation before Plan, and clean v1 as the compatibility
baseline without automatic POC migration. The Specification was explicitly
approved on 2026-09-20. The authorized reconciliation records
[ADR-0005](../decisions/0005-bounded-local-filesystem-threat-model.md) for HD-3,
[ADR-0006](../decisions/0006-machine-local-detail-artifacts.md) for HD-2, and H13
updates across Specification 002. PR #70 merged that reconciliation and received
explicit human approval. The resulting
[Specification 004 Plan](../specifications/004-mvp-v1-baseline/plan.md) is ready
for human review. Review identified
[ADR-0007](../decisions/0007-local-publication-and-recovery-protocol.md), proposing
the shared local publication/recovery protocol, and
[ADR-0008](../decisions/0008-minimal-machine-local-execution-record.md), proposing
the bounded machine-local sequential Execution record. Both remain Proposed and
must be explicitly accepted or revised before Plan approval/Tasks. The Plan does
not authorize implementation, Provider mutation, or release.

## Complete Specification 002

[Specification 002 — Lingo Project Initialization](../specifications/002-lingo-project-initialization/spec.md)
is the first active vertical slice. Its approved scope connects minimal Project
creation, explicit update, validation, reopening and local installation.
The POC covers only the minimal one-file path. Completing Specification 002 still
requires separately authorized delivery of its remaining application, security,
recovery and acceptance contracts under the
[Plan](../specifications/002-lingo-project-initialization/plan.md).

Minimal init requires identity and schema version, without mandatory Repository,
Runtime or Provider configuration. Update must materialize and validate complete
proposed state before publication. Portable working copy and ID-addressed local
state remain separate; operational installation must prove the safety and recovery
contracts, not merely serialize valid data.

Git backing remains optional and independent of Repository association. Project
mutation, optional local commit and explicitly authorized remote push/sync are
separate operations. Concrete Git execution and synchronization are outside this
slice's delivery; the authority boundary does not select their mechanisms.

## Later capabilities requiring specification

The following possibilities have no approved delivery order. Each needs a bounded
Specification and explicit authority before implementation; dependencies explain
what must be understood first.

| Capability | Dependency or question to resolve |
|---|---|
| Runtime and model discovery | Adapter lifecycle, capability reporting and failure semantics beyond portable declarations |
| Capability negotiation and Integration bootstrap | Workflow needs, provider-neutral capability vocabulary, explicit setup authority and credential-source resolution |
| Agent Planning and orchestration | Execution graphs, dependencies, parent/child Executions, approval/failure handling and durable Evidence contracts |
| Additional Runtime skills and adapters | Validate a second Runtime before generalizing the delivered Codex-only boundary |
| Richer Project wizard | Runtime/Integration discovery and setup contracts beyond the minimal guided flow already specified |
| Portable distribution and synchronization | Independent local/remote Git authority, conflict handling, recovery and explicit compatibility policy |

Provider choices remain independent; `Role != Model`, `Execution != Agent`,
`Provider != Transport` and `Integration != MCP`. A selected declaration never
proves connectivity or operational readiness. Evidence and human prioritization
may combine, reorder, split or defer these capabilities.
