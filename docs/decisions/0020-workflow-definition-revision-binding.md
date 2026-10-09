# ADR-0020 — Workflow definition ownership and revision binding

## Status

Accepted on 2026-10-09 by the Axiom maintainer, explicitly recorded in
[Issue #271](https://github.com/rgomids/axiom/issues/271#issuecomment-6074127314).
The accepted proposal is bound to reviewed main revision
`998e763cf82c4cb443d0d4d0051c6e82bcdf8d72` (ADR blob
`3f913ed027ec90546ca96d436d538335590e4279`) and Specification 007 blob
`b68072e9909793f24786c343e4b9d3b83064b217`.
Acceptance of these contracts does not itself authorize dependent implementation,
Runtime dispatch, merge, publication or MVP acceptance.

## Context

The expanded MVP needs editable Project workflows whose running/historical
Executions retain their original semantics. A mutable name or workflowVersion
alone cannot prove what instructions, gates, validators or agents were authorized.
Ownership, snapshots, cleanup and rework affect durable public/storage contracts
and cross the architecture-policy ADR threshold.

## Decision

Adopt Project-owned portable companion definitions, explicit version-gated
Project selection, immutable revisions identified by validated JCS/SHA-256 content,
and a complete machine-local snapshot pinned before new Execution admission.
Adopt new linked Executions for human-rejected delivery rework. The complete
proposed behavior/DTOs/acceptance matrix live in
[Specification 007](../specifications/007-configurable-workflows/spec.md).

Project definitions are portable intent, not Repository ownership or skill state.
Execution snapshots, local context/artifacts, runtime observations, authority and
history remain machine-local. Edits/select R2 affect only subsequent starts; resume
uses retained R1. Unknown/missing/tampered binding fails closed. Cleanup is
explicit, reference-aware and cannot prune needed execution/review/recovery state.
The bounded portable revision index retains retired identity/digest assignments;
removal never permits reassignment of an old revision to new content.
No general workflow engine, shared catalog, new control plane or migration by
inference is introduced. Authoring stays a Project operation; execution stays
under the Work Item domain surface.

### Exact relationship to accepted decisions

- [ADR-0003](0003-lingo-as-axiom-local-control-plane.md): unchanged. Domain/application
  own rules; Lingo coordinates; skills are thin calls.
- [ADR-0004](0004-portable-project-manifest.md): extends portable intent with
  version-gated selection and companion content; immutable Project identity and
  portable/local/credential separation remain intact. No accepted fragment is replaced.
- [ADR-0008](0008-minimal-machine-local-execution-record.md): remains authoritative
  for all legacy format-1 sequential records and their unchanged resume. New
  configurable records use a separate format and binding; no synthetic migration.
- [ADR-0009](0009-parent-child-execution-graph.md): remains authoritative for graph
  lineage, authority, dependencies, isolation, integration, attempts and coordination.
  Stage compilation adds correlation, not a second scheduler. A one-agent stage
  may use the sequential executor and does not invent a graph/integration agent.
- ADR-0005/0006/0007 continue to own filesystem confinement, artifact retention and
  publication/recovery. This proposal cannot strengthen unsupported durability claims.

This is an additive accepted decision, not supersession. It amends no earlier
accepted ADR text or historical Txx/Evidence. Any later contradictory change
requires a new explicitly accepted decision following ADR-0018; this acceptance
does not retroactively alter existing ADRs.

## Alternatives considered

| Option | Complexity/coupling/operability | Security/cost/portability | Reversibility/migration |
|---|---|---|---|
| Companion definitions + retained snapshot (recommended) | Reuses Project/Execution authority; extra references and codecs | Local bounded storage cost; portable intent, no paid service; offline resume | New versioned formats; old records stay intact; immutable snapshots constrain cleanup |
| Inline all revisions in Project manifest | Simple single-file transfer; growing mutation/read payload | Portable but larger publication/validation blast radius | Later splitting changes storage/public contracts |
| Global mutable workflow catalog | Reuse across Projects; new catalog owner/resolver and availability dependency | Cross-Project authority ambiguity; local/global drift | Requires catalog lifetime and explicit migrations |
| Pin only a digest/reference | Small Execution record | Missing definition prevents offline resume; source availability becomes critical | Later snapshots require recovery of exact historical content |

Exact source-byte hashing is simpler than JCS but treats formatting changes as
new semantics. An implementation-specific marshaler couples public hashes to a
client/language. JCS adds conformance work but provides an independently specified
wire identity without changing existing graph codecs.

For rework, rewinding stages or replacing completed attempts uses fewer records
but makes prior acceptance/authority/artifact interpretation ambiguous. A new
linked Execution costs storage and repeated gates, while preserving rejected
history and requiring fresh scope/revision/effect authority.

## Consequences

### Positive

- R1/R2 independence, deterministic preview identity and offline recovery are reviewable.
- Existing domain, graph and skill boundaries stay coherent; no mandatory multi-agent stage.
- Historical records remain readable without invented facts or auto-migration.

### Negative / trade-offs

- #273 owns a new portable definition/schema and JCS conformance; #274 owns new local
  format/inventory/corpus and atomic snapshot admission; retained content consumes space.
- Companion references require import/read-back checks. Reference-aware cleanup can
  refuse deletion when state is uncertain. Rework repeats gates under fresh authority.
- #275/#276 must carry stage context/control correlation without weakening graph authority.

## Human decision and downstream gate

Specification HD-001 (Project-owned companion definitions and local immutable
snapshots), HD-002 (Project authoring + Work Item execution), HD-003 (new linked
Execution on rejection) and HD-004 (RFC 8785 JCS + SHA-256) were expressly
accepted without additional conditions by the Axiom maintainer on 2026-10-09.
Authority and exact reviewed revision are recorded in
[Issue #271](https://github.com/rgomids/axiom/issues/271#issuecomment-6074127314).
Acceptance is separate from implementation authorization and operational Evidence;
#273–#278 retain their individual gates and scope.

## Revisit when

Cross-Project reuse, remote synchronization, larger workflow structures, installed
adapter constraints or bounded retention evidence invalidates these trade-offs.
Reconsideration needs human decision and explicit compatibility/authority impact.
