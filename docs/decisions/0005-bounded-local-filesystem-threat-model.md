# ADR-0005 — Bounded local filesystem threat model

## Status

Accepted on 2026-09-20 as the direct architectural consequence of the human
[HD-3 decision](../specifications/004-mvp-v1-baseline/spec.md#hd-3--mvp-filesystem-threat-model)
recorded during review of Specification 004. This ADR defines the proof boundary;
it does not authorize the Specification 004 Plan, Tasks, implementation, release,
or a particular filesystem mechanism.

## Context

Specification 002 originally required that no Lingo write escape or be redirected
from the exact authorized target under traversal, links, races and TOCTOU. Its
acceptance matrix intentionally left mechanisms to implementation. Without an
explicit adversary and durability boundary, that wording could be read as requiring
proof against every hostile interleaving available to another process with the
same operating-system user identity and as requiring survival across physical
power loss or media failure.

The POC added private roots, anchored operations, no-follow and no-replace
publication, link and ACL checks, process coordination, expected revisions,
attempt markers, injected faults, and manual recovery. Linux/macOS Evidence showed
useful protection against controlled traversal, link, replacement, concurrency,
crash-boundary and storage-fault cases. It did not prove arbitrary malicious
same-UID interleavings, physical power-loss durability, every filesystem fault,
or physical-media durability. The POC was accepted with those limitations as
historical Evidence, not as proof of the stronger interpretation.

The MVP is local-first and single-user. Requiring a defense against an actor with
the same effective identity that can continuously mutate every object between
checks would impose a materially different security and storage architecture.
Physical power-loss and media-durability guarantees would likewise require an
explicit hardware/filesystem support matrix and destructive testing beyond the
MVP outcome. HD-3 therefore narrows the proof boundary while preserving the
properties required for safe supported local operation.

## Decision

Axiom adopts a **bounded local filesystem threat model** for the MVP and for the
reconciled completion boundary of Specification 002.

### Supported environment and assumptions

- Operations run as the intended local user on a declared supported OS/filesystem
  combination whose required primitives and semantics have been validated.
- Axiom processes may overlap, fail, be interrupted, or present stale observations.
  They are not trusted to coordinate correctly without application-enforced
  locks, expected revisions, or equivalent process-concurrency controls.
- Inputs, portable references, existing directory entries, links, ownership,
  permissions, and leftover operation artifacts are untrusted.
- No malicious same-UID actor continuously performs arbitrary hostile mutations
  between every supported validation and commit boundary.

If a required supported assumption or property cannot be established, the
operation fails closed before mutation or returns `recovery_required` without
claiming rollback or success.

### Supported threats and mandatory properties

1. **Authorized-target confinement.** Every write, replacement, move, cleanup,
   quarantine, and recovery mutation remains within the exact root/object set
   covered by current authority. Lexical similarity, caller CWD, or a previously
   resolved path never grants authority.
2. **Traversal protection.** Absolute, escaping, ambiguous, or otherwise
   non-contained portable references and relative names are rejected before use.
3. **Supported link and replacement attacks.** Pre-existing symlink/hard-link
   targets, unsafe link counts/types, changed directory or leaf identity, and
   controlled link/ancestor/leaf replacement at the implementation's declared
   inspection and commit boundaries must be detected and rejected. Read-only
   binding resolution grants no write authority.
4. **Process concurrency.** Conflicting Axiom processes cannot both commit from
   stale authority. Readers never accept a mixed or invalid canonical Project.
   Coordination, expected revisions, object identity checks, or equivalent
   mechanisms must cover create, install, update, rename, cleanup, and recovery.
5. **Deterministic injected faults.** The future Plan must enumerate the concrete
   commit protocol and inject failures at its material stages, including incomplete
   staging/write, pre-publication, publication, post-publication/pre-acknowledgment,
   and cleanup. Short writes, space/quota errors, permission denial, interruption,
   and applicable rename/sync errors are classified without invented success.
6. **Partial publication protection.** Canonical readers observe either the prior
   complete valid state or the new complete valid state. A partial, mixed, malformed,
   or marker-ambiguous state is never accepted as canonical.
7. **Truthful atomicity.** Atomicity means the logical old-or-new contract at the
   documented commit point, not a universal physical transaction. Pre-commit
   failure preserves prior authority. A confirmed post-commit effect remains
   committed even when later local state, projection, sync, or cleanup fails.
8. **Fail-closed uncertainty and guided recovery.** Recognized interrupted state
   produces deterministic `recovery_required`; unknown artifacts are preserved.
   Recovery mutates only positively identified, authorized objects after explicit
   inspection. Ambiguity requires operator review and never automatic deletion.
9. **Ownership and permissions.** Machine-local state, staging, recovery metadata,
   and detail artifacts use restrictive platform-appropriate ownership,
   permissions, and supported ACL checks. Unsafe observed conditions fail closed;
   portable export excludes local content by construction.
10. **Bounded inputs and artifacts.** Filesystem reads, writes, captured output,
    recovery metadata, and generated artifacts are size-bounded and sanitized.

The future Specification 004 Plan may select concrete mechanisms only after it
maps these properties to the declared OS/architecture/filesystem support matrix.
Syscalls, lock layout, filenames, Go packages, and staging topology remain deferred.

### Explicit exclusions

The MVP makes no guarantee against:

- a malicious process running under the same UID that performs arbitrary hostile
  interleavings outside the declared supported inspection/commit boundaries;
- survival, acknowledgment durability, or deterministic recovery across physical
  power loss;
- physical-media durability, latent media corruption, controller failure, or
  storage hardware lying about persistence.

These are unsupported threats/guarantees, not risks claimed as solved. Observed
state after any excluded event is still treated conservatively: invalid or
ambiguous state fails closed and requires operator review.

## Alternatives considered

### Retain the unbounded interpretation

Strongest reading of the original wording, but not credibly testable within the
MVP and likely to require privilege separation, a trusted helper, different UID,
or a different storage boundary. It would keep the Plan blocked on guarantees not
needed for the accepted local journey.

### Accept the POC mechanisms as the permanent contract

Fastest path, but converts historical experiments and concrete implementation
choices into architecture without evaluating the complete MVP. Rejected: the POC
is Evidence, and future planning still owns mechanisms and the exact fault matrix.

### Bounded local threat model

Preserves deterministic confinement, concurrency, complete-state publication,
fault injection, truthful commit semantics, and recovery while stating which
adversaries and durability claims are unsupported. This is the HD-3 decision.

## Consequences

### Positive

- Specification 002 and Specification 004 share one explicit proof boundary.
- Plan and acceptance Evidence can target reproducible Linux/macOS behavior rather
  than imply unbounded adversarial or hardware guarantees.
- Existing mandatory safeguards remain normative; narrowing the adversary does
  not permit traversal, unsafe links, stale concurrent commits, partial canonical
  state, silent cleanup, or false rollback.
- POC Evidence remains useful without becoming the permanent implementation contract.

### Negative / trade-offs

- Same-UID isolation is not a security boundary for the MVP. Users must not treat
  local Axiom state as protected from arbitrary code running as that same user.
- A successful operation does not promise persistence through physical power loss
  or media failure.
- The future Plan must declare supported filesystem assumptions, material fault
  stages, and exact Evidence; undeclared combinations remain unsupported.
- Specification 002 SEC-003/SEC-005, AC-07/09/14/16, its Plan, Tasks, and Evidence
  need dated reconciliation rather than silent deletion of their original history.

## Revisit when

- a multi-user, daemon, privileged-helper, or cloud control-plane boundary is
  specified;
- Axiom stores data whose loss or corruption requires power-loss or media-level
  durability guarantees;
- a supported environment cannot provide the required confinement, concurrency,
  ownership, or logical publication properties;
- real incidents or Evidence show that the bounded same-UID assumption is unsafe
  for the intended deployment.

Reconsideration requires a new threat analysis, explicit human decision, updated
support matrix, migration/rollback impact, and executable Evidence.
