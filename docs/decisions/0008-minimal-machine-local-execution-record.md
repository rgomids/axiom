# ADR-0008 — Minimal machine-local Execution record

## Status

**Proposed — ready for human review.**

This proposal was identified during review of the
[Specification 004 Plan](../specifications/004-mvp-v1-baseline/plan.md). It is
not Accepted and does not authorize Tasks, implementation, Provider projection,
workflow execution, migration, or release. Human acceptance or explicit revision
is required before the Plan can advance.

## Decision in one sentence

The MVP should persist one minimal, versioned, machine-local Execution record as
the authoritative resume and transition record for its single sequential workflow,
without defining Axiom's future general Execution graph.

## Context

Specification 004 requires truthful local workflow stages, interruption/resume,
expected revisions, Provider projection, completion, provenance, artifacts, and
Evidence references. The Plan proposes a machine-local record containing Execution
identity, Project and Repository scope, Work Item and Runtime references, workflow
version, current stage, transitions, revision, timestamps, provenance, references,
and terminal state.

This is more than a local codec detail. The record crosses workflow, Provider
projection, Runtime invocation, completion, artifact/Evidence correlation,
recovery, and compatibility. Its authority and identity become expensive to
change after Provider projection keys and retained Evidence refer to it.

[ADR-0003](0003-lingo-as-axiom-local-control-plane.md) places executable workflow
and Execution state in Lingo while preserving Axiom's domain ownership.
[ADR-0006](0006-machine-local-detail-artifacts.md) makes Execution the primary
correlation when one exists and rejects a parallel `operation-attempt` domain
entity. Neither decision defines the bounded MVP record proposed here.

## Problem

The MVP cannot resume or reconcile a transition truthfully from mutable Provider
labels, Runtime chat, artifacts, or current-stage text alone. It needs a stable,
revisioned local authority, but adopting the future general Execution graph would
expand scope beyond the approved single sequential workflow.

## Constraints

- `Execution != Agent`; a record must not describe an Agent identity or general
  orchestration graph.
- Local validated workflow state remains authoritative; Provider state is a
  separately authorized projection.
- Project remains distinct from Repository, and portable Project intent must not
  contain Execution state.
- Runtime skills remain thin and cannot own another workflow state model.
- Detail artifacts are subordinate to Execution when one exists under ADR-0006.
- Clean v1 is the compatibility baseline; no automatic POC migration is implied.

## Proposed decision

### Architectural invariants

1. **One bounded record per MVP Execution.** The record represents one run/resume
   lineage of the approved sequential workflow. It is not a general DAG, scheduler,
   Agent plan, or event-sourcing platform.
2. **Stable opaque identity.** Execution identity is allocated once, remains stable
   across resume and Provider reconciliation, and is not derived from Project,
   Repository, Work Item, Runtime, path, or Provider identity.
3. **Machine-local authority.** The record is Axiom-owned machine-local state. It
   never enters portable Project intent or Repository content by convenience.
4. **Explicit scope references.** It references Project ID, one Project-scoped
   Repository key when applicable, the exact Work Item reference, and Runtime ID
   without collapsing those concepts or making a Provider/Runtime a domain owner.
5. **Versioned workflow contract.** It records workflow/format compatibility and
   the current gate from the approved sequential gate set. Unsupported or malformed
   versions fail closed before transition.
6. **Revisioned transitions.** Each committed transition advances one expected
   state revision and records prior/current stage, bounded outcome, required
   authority/effect facts, provenance, time, and relevant artifact/Evidence
   references. Resume reads the latest committed revision; it never infers success
   from chat, Provider labels, or an interrupted process.
7. **Terminal truth.** Terminal state records the canonical completion meaning and
   confirmed effects. A later projection, rendering, artifact, or cleanup failure
   cannot erase a confirmed local transition or invent rollback.
8. **Projection correlation, not ownership.** Provider projection keys may derive
   from stable Execution identity plus committed transition revision. Provider
   metadata cannot create, advance, or repair local Execution truth.
9. **Bounded history.** The record retains enough transition history for resume,
   reconciliation, review, and Evidence traceability. It is not raw chat history,
   unrestricted model reasoning, or unbounded external output.
10. **Shared local publication contract.** Mutation follows the accepted local
    publication/recovery decision if ADR-0007 is accepted. Until then, concrete
    persistence remains blocked.

### Minimum semantic fields

The MVP record needs these semantic categories:

- opaque Execution ID;
- Project ID and applicable Repository key;
- exact Work Item reference and Runtime ID;
- record/workflow version;
- current stage and expected state revision;
- bounded transition records;
- creation/update/transition timestamps;
- canonical provenance;
- artifact and Evidence references;
- terminal state and confirmed-effect summary when terminal.

Exact field names, JSON/YAML/binary representation, path/layout, timestamp format,
transition encoding, package names, indexes, and size values remain Plan/Task and
implementation details. Adding a field is not automatically architectural; a
change becomes architectural when it changes identity, authority, lifecycle,
compatibility, or cross-boundary meaning.

## Alternatives considered

### Reconstruct workflow from Provider state and artifacts

Avoids a dedicated record, but makes an external projection authoritative, cannot
reliably distinguish interruption from completion, and couples resume to Provider
availability and mutable comments/labels.

### Persist only current stage

Smallest storage shape, but lacks expected-revision history, confirmed-effect
truth, idempotent projection correlation, and inspectable resume/Evidence context.
Later expansion would change identity and compatibility after release.

### Implement a general Execution graph now

Could support future orchestration, dependencies, parent/child runs, and multiple
agents. Those capabilities are explicitly outside the MVP and lack approved
requirements. This would create premature architecture and migration cost.

### Minimal versioned sequential Execution record

Supports the approved workflow, resume, projection, completion, and Evidence while
leaving general graph/orchestration semantics open. This is the proposed option.

## Consequences

### Positive

- Local resume and transition truth no longer depend on Runtime chat or Provider
  metadata.
- Execution provides one stable correlation for artifacts, Evidence, completion,
  and Provider projection.
- Revisioned transitions support stale-authority and idempotency checks.
- The broad future Execution model remains open.

### Negative / trade-offs

- Axiom owns another versioned local format and compatibility boundary.
- Transition history consumes bounded local capacity and needs retention/recovery
  rules coordinated with referenced artifacts/Evidence.
- The sequential shape does not directly support multi-agent or DAG orchestration.
- Changing identity, stage semantics, or projection correlation after release may
  require migration and Provider reconciliation.

## Risks and mitigations

- **Premature generalization:** prohibit graph, scheduler, Agent ownership, and
  universal event-log abstractions in the MVP record.
- **Schema overcommitment:** keep exact wire field names and layout in later
  schema/implementation work while fixing only cross-boundary semantics here.
- **Unbounded history:** bound record/transition sizes and reference external
  detail artifacts rather than embedding logs or chat.
- **Provider coupling:** store exact provider-neutral Work Item references and keep
  projection adapter-specific and reconcilable.
- **Portable leakage:** enforce machine-local ownership by construction, not only
  ignore rules.

## Reversibility and migration impact

Before release, the proposal can be rejected or revised without user-state
migration. After Execution IDs appear in artifact metadata, Evidence, and Provider
projection keys, changing identity or revision semantics becomes expensive and may
require versioned local migration plus external reconciliation. Exact encoding,
paths, and in-process types remain more reversible if semantic identity and
authority are preserved.

## Revisit when

- an approved multi-agent or dependency-graph workflow requires parent/child or
  graph semantics;
- every artifact-producing operation gains an Execution and ADR-0006 correlation
  fallback can be simplified;
- remote collaboration or synchronization requires an explicitly portable or
  shared Execution representation;
- Evidence shows the bounded transition history is insufficient or too costly;
- another local authority model can preserve resume/projection truth with less
  complexity.

Reconsideration requires explicit human review, compatibility and Provider
reconciliation impact, and executable Evidence. Validation or merge does not
accept this proposal.
