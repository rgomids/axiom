# ADR-0007 — Local publication and recovery protocol

## Status

**Proposed — ready for human review.**

This proposal was identified during review of the
[Specification 004 Plan](../specifications/004-mvp-v1-baseline/plan.md). It is
not Accepted and does not authorize Tasks, implementation, migration, cleanup,
release, or any local or Provider mutation. The Plan cannot be approved or used
to advance to Tasks until a human accepts, rejects, or explicitly revises this
decision.

## Decision in one sentence

All mutable Axiom-owned local stores should implement one shared logical
publication and recovery protocol that preserves prior-or-new complete authority,
uses deterministic cross-store coordination, exposes one truthful commit point,
and fails closed while publication state is intermediate or ambiguous.

## Context

Specification 004 requires versioned local workflow state, Project installation
state, detail artifacts, references, release receipts, compatibility records, and
recovery metadata. Several operations can also update more than one related local
object. If each store chooses independent commit, reader, concurrency, and recovery
semantics, callers cannot classify confirmed effects consistently and cross-store
operations can deadlock, expose mixed state, or invent rollback.

[ADR-0005](0005-bounded-local-filesystem-threat-model.md) already defines the
supported adversary and durability boundary. It requires exact-target confinement,
supported link/replacement protection, process concurrency, deterministic injected
faults, complete canonical state, truthful pre/post-commit outcomes, restrictive
metadata, and guided recovery. It explicitly excludes arbitrary malicious same-UID
interleavings and physical power-loss/media durability. This ADR consumes that
boundary. It neither broadens it nor repeats it as a new threat model.

[ADR-0004](0004-portable-project-manifest.md) requires logical old-or-new Project
publication but deliberately leaves the persistence mechanism open. ADR-0006
requires atomic validity and safe cleanup for detail artifacts. The proposed MVP
Plan currently selects shared coordination, staged publication, protected
replacement, multi-object recovery state, a commit point, prior/new generations,
and reader behavior. Those choices are cross-cutting, durable, and expensive to
change after formats and recovery tools ship, so they cannot remain implicit in a
Plan.

## Problem

Axiom needs one reviewable answer to three coupled questions: when a local mutation
becomes authoritative, what readers may return while publication is interrupted,
and how recovery distinguishes prior, committed, and ambiguous state. Leaving
those answers per-store would make completion truth and recovery incompatible
across Project, workflow, artifact, receipt, and other mutable local records.

## Constraints

- Preserve exact authority, expected-revision, portable/local, and confirmed-effect
  contracts already approved by Specifications 002 and 004.
- Remain within ADR-0005's supported threat and durability boundary.
- Support single-object create/update and bounded multi-object publication without
  claiming a general database transaction.
- Keep readers deterministic during interruption and recovery.
- Avoid turning syscall names, filenames, directory layouts, lock libraries, or Go
  packages into architecture.
- Permit later storage-adapter replacement without changing user-visible authority
  or completion semantics.

## Proposed decision

### Architectural invariants

1. **One logical protocol.** Mutable Axiom-owned local stores share the same
   publication states and outcome meanings: inspect, coordinate, prepare, validate,
   publish, confirm, and clean up. An adapter may use different mechanisms only if
   it preserves the same observable invariants and fault classifications.
2. **Explicit authority and revision.** Mutation binds exact objects, intended
   effects, preview, and expected revisions. Any changed observation invalidates
   stale authority before commit.
3. **Deterministic coordination.** Operations acquire all required coordination
   scopes in one documented broad-to-narrow total order. No store may introduce a
   competing order. Exact lock objects and operating-system APIs are implementation
   details.
4. **Private preparation.** New content is prepared outside canonical visibility,
   within the same supported local filesystem publication boundary, with restrictive
   ownership and permissions. It is complete, bounded, decoded, and revalidated
   before it can become canonical.
5. **Protected publication.** Create never replaces an existing canonical object.
   Update or relocation replaces only the exact observed generation under current
   authority. Unsupported cross-filesystem publication fails before mutation.
6. **Bounded multi-object recovery record.** An operation that cannot publish as
   one canonical object records enough versioned, positively owned state to
   distinguish intended objects, protocol stage, expected revisions, and prior/new
   complete generations. This is a recovery protocol, not a claim of physical or
   distributed transactionality.
7. **Single logical commit point.** Commit occurs when the complete new canonical
   generation has been published under valid authority. Before that point, the
   prior complete generation remains authoritative. After a confirmed commit, the
   new generation remains authoritative even when acknowledgment, Provider
   projection, secondary local persistence, or cleanup fails.
8. **Prior/new generation semantics.** Recovery may select or restore only a
   completely validated generation identified by the owned recovery state. It may
   not synthesize a mixed generation or treat absence as proof that rollback
   happened.
9. **Fail-closed readers.** Readers validate format, object identity, revision, and
   relevant recovery state before returning canonical content. Recognized
   intermediate or uncertain state returns `recovery_required`; contradictory or
   unknown state is preserved for operator review.
10. **Owned-only cleanup.** Cleanup occurs after commit classification and removes
    only exact, positively identified protocol objects. Cleanup failure cannot
    change commit truth.

### Implementation details deliberately deferred

The Plan and later authorized Tasks may choose and validate:

- concrete syscalls and platform-specific fallbacks;
- advisory-lock implementation, lock filenames, and lock storage;
- staging, generation, marker, and recovery-record names and layouts;
- journal encoding and whether one or multiple files represent it;
- Go packages, interfaces, libraries, and error types;
- bounded retry/backoff and cancellation plumbing;
- flush/sync calls needed to surface reported I/O errors inside ADR-0005's explicit
  exclusion of physical power-loss/media durability;
- optimizations that preserve the invariants above.

Those choices require native Evidence on every supported OS/architecture/filesystem
row. They do not become architecture merely because one implementation uses them.

## Alternatives considered

### Independent protocol per store

Reduces initial shared design, but duplicates commit/outcome rules, permits lock
order conflicts, and makes recovery and completion classification store-specific.
Cross-store operations become harder to reason about and migrate.

### One embedded transactional database for all local state

Provides mature transactions and recovery for records within one database. It
would centralize ownership and migration, but portable working-copy files, installed
binary/skill targets, and filesystem artifacts still cross that boundary. Adopting
a database now would add a durable dependency and would not eliminate external
filesystem publication.

### Best-effort staged writes without durable recovery state

Simpler during normal operation, but interruption across related objects leaves no
authoritative way to distinguish prior, committed, and partial effects. Readers
would need heuristics that conflict with fail-closed recovery.

### Shared logical protocol with adapter-specific mechanisms

Creates one semantic contract without freezing syscalls, filenames, or a storage
engine. It increases up-front design and testing cost but keeps commit truth,
coordination, readers, and recovery coherent. This is the proposed option.

## Consequences

### Positive

- Completion can classify pre-commit, committed, partial, and uncertain outcomes
  consistently across local stores.
- Cross-store lock ordering and recovery behavior become reviewable invariants.
- Readers never silently combine generations or accept an interrupted state.
- Concrete platform mechanisms remain replaceable behind one semantic boundary.
- ADR-0005 remains the single threat-model authority.

### Negative / trade-offs

- Every mutable local adapter must conform to a shared state machine and fault
  vocabulary even when a store could use a simpler private mechanism.
- Multi-object operations require bounded recovery metadata and operator tooling.
- Fail-closed readers may reduce availability until recovery is explicitly reviewed.
- Format/protocol versioning and future migrations add maintenance cost.
- The protocol provides logical old-or-new truth, not physical durability or a
  distributed transaction.

## Risks and mitigations

- **Over-generalization:** keep the protocol limited to Axiom-owned mutable local
  state required by the approved MVP; do not expose a universal storage framework.
- **Mechanism leakage:** keep syscalls, paths, packages, and encodings in Plan/Tasks
  and implementation Evidence.
- **Recovery metadata corruption:** strictly version, bound, validate, and preserve
  uncertain state; never infer cleanup authority.
- **Deadlock or starvation:** require one total coordination order and native
  multi-process Evidence; exact scheduling policy remains implementation detail.
- **False durability claims:** report only logical commit truth allowed by ADR-0005.

## Reversibility and migration impact

Before any MVP persisted format is released, this proposal is reversible through
human revision of the Plan and ADR with no user-state migration. After release,
changing commit points, reader behavior, generation identity, or recovery records
can require versioned compatibility, explicit migration/rollback design, and new
native Evidence. Replacing a lock or syscall implementation remains comparatively
reversible when observable invariants and persisted formats do not change.

## Revisit when

- a single transactional store can encompass all required local mutation without
  weakening portable-file and install-target boundaries;
- a supported filesystem cannot provide the primitives needed for the invariants;
- multi-user, daemon, remote, or distributed coordination enters approved scope;
- measured operation/recovery cost makes the shared protocol disproportionate;
- incidents or Evidence invalidate the selected commit point or reader behavior.

Reconsideration requires explicit human review, compatibility/migration impact,
and updated Evidence. This proposal must not be treated as Accepted by merge,
validation, or Plan approval alone.
