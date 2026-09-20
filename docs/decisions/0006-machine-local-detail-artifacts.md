# ADR-0006 — Machine-local detail artifacts

## Status

Accepted on 2026-09-20 as the direct architectural consequence of the human
[HD-2 decision](../specifications/004-mvp-v1-baseline/spec.md#hd-2--detailed-artifact-ownership-and-lifecycle)
recorded during review of Specification 004. This ADR defines ownership and
lifecycle invariants. It does not authorize the Specification 004 Plan, Tasks,
implementation, or a concrete filesystem layout.

## Context

Specification 004 requires concise completion output. Diagnostics, review context,
recovery guidance, validation matrices, and durable Evidence support may exceed
that summary and therefore need referenced Markdown detail artifacts.

Those artifacts cannot enter the Portable Project Manifest or working copy:
they may contain machine paths, local observations, failure context, workflow
state, and other non-portable data. They also cannot be temporary-by-default;
workflow continuation, review, recovery, and Evidence may depend on them after the
originating command ends.

Execution is the candidate durable record for a bounded attempt, but some commands
fail validation, installation, or setup before an Execution exists. Creating an
`operation-attempt` domain entity solely to name those artifacts would add a
parallel lifecycle without a demonstrated need. HD-2 instead requires durable
machine-local artifacts, safe explicit cleanup, Execution-aware correlation, and
a non-domain pre-Execution correlation identity.

## Decision

### Ownership and storage boundary

Axiom owns detail artifacts through one machine-local artifact-management boundary.
That boundary controls creation, identity, lookup, retention metadata, references,
and cleanup. Runtime skills, renderers, Providers, Projects, Repositories, and
individual commands do not invent separate ownership or retention rules.

Artifacts live under a dedicated Axiom-managed machine-local root, separate from:

- the Portable Project Manifest and portable Project working copy;
- associated Repository working copies;
- installation records and their schema authority;
- external credential stores.

The exact OS path, directory/filename layout, storage adapter, and Go package are
future Plan/implementation decisions. A Project or Repository reference scopes and
helps discover an artifact; it does not make that Project/Repository the storage
owner or permit portable publication.

### Identity and correlation

Each artifact has an immutable, opaque, stable artifact identity independent of
its display filename and physical location. A display name is not authority and
must not be used as the sole lookup key.

When an Execution exists, the artifact is subordinate to and references that
Execution. Execution provides the primary operational correlation; the artifact
does not create a competing attempt model.

Before an Execution exists, an operation may allocate an opaque machine-local
correlation identity. That identity exists only to correlate the operation's
summary, artifact, logs/events, recovery context, and later Evidence reference.
It is not a Project identity, Work Item, Execution, domain entity, authority token,
or portable identifier. If the work later creates an Execution, the existing
artifact may add the Execution reference without changing its artifact identity
or erasing the original local correlation.

### Required metadata and Evidence relationship

Every artifact records, within bounded sanitized metadata or content:

- stable artifact identity;
- applicable Execution identity or pre-Execution local correlation identity;
- creation time and canonical Axiom provenance;
- relevant Project, Repository, Work Item, Provider, and source references when
  known and safe;
- outcome/category and retention class;
- content size and integrity digest suitable for later reference;
- relationship to any superseding artifact or cleanup state when applicable.

A detail artifact is not Evidence merely because it exists. Evidence makes an
inspectable claim and references the artifact identity plus the relevant digest,
source, observation, and verification context. Diagnostic context remains distinct
from Evidence claims. An artifact required by retained Evidence cannot be removed
while that reference is live under its retention policy.

### Lifecycle and retention

Artifact lifecycle is:

```text
created -> retained -> eligible-for-cleanup -> removed
                    \-> preserved-for-review
```

Creation is atomic with respect to artifact validity: incomplete content is not
returned as a usable detail reference. Failure to create a detail artifact must
not hide or rewrite the primary operation's confirmed outcome.

Retention is purpose-based, not an accidental side effect of filename age. The
future Plan may define concrete durations, but must preserve these classes:

- active workflow/Execution or recovery dependency;
- retained Evidence dependency;
- bounded diagnostic artifact with no live workflow/Evidence dependency;
- preserved-for-review because ownership, validity, or references are uncertain.

Artifacts remain durable while any active workflow, Execution, recovery procedure,
human review, or retained Evidence depends on them. Eligibility can be computed
only from authoritative local references and the applicable retention policy.

### Safe cleanup

Cleanup is an explicit, separately authorized operation. It must:

1. preview exact artifact identities, purpose, size, retention basis, and reason
   each artifact is eligible;
2. refuse artifacts referenced by an active workflow/Execution, unresolved
   recovery state, retained Evidence, or another retained artifact whose contract
   requires the reference;
3. revalidate identity, ownership, type, link state, and eligibility immediately
   within the supported filesystem boundary before removal;
4. remove only positively identified Axiom-owned artifacts, never unknown content
   or an entire broad root by recursive assumption;
5. report partial/uncertain cleanup truthfully, preserve unknown state, and require
   operator review rather than infer success;
6. retain a bounded, sanitized cleanup record sufficient to explain what was
   removed without retaining the removed sensitive payload itself.

Automatic age-based deletion, incidental cleanup after another operation, and
cleanup by deleting the whole root are not authorized by this decision.

### Security and size

- Secrets, credential values, raw chat history, unrestricted model reasoning, and
  unbounded external output are forbidden.
- Content and captured output are bounded before publication. Exact per-type and
  aggregate limits belong to the future Plan.
- Artifacts use restrictive platform-appropriate ownership, permissions, supported
  ACL/link checks, and the bounded local threat model in
  [ADR-0005](0005-bounded-local-filesystem-threat-model.md).
- Sanitization failure, uncertain ownership/reference state, or inability to
  prove safe cleanup fails closed and preserves the artifact for review.
- Publication outside the machine-local boundary requires a separate explicit
  action, authority, sanitization/review contract, and destination-specific
  Evidence. A detail reference alone grants no publication authority.

## Alternatives considered

### Put details in the Portable Project working copy

Easy discovery and versioning, but leaks machine-local context, pollutes portable
intent, and couples cleanup to Git. Rejected by HD-2 and ADR-0004.

### Keep details only as ephemeral command output

Simple lifecycle, but loses recovery, continuation, review, and Evidence context.
Rejected because HD-2 requires durability while needed.

### Introduce an `operation-attempt` domain entity

Provides a uniform parent before Execution, but duplicates candidate Execution
semantics and creates a new permanent conceptual lifecycle without proven need.
Rejected for the MVP. Local pre-Execution correlation supplies identity without
domain expansion.

### Execution-only ownership

Clear once execution starts, but cannot represent setup, validation, installation,
or other failures before an Execution exists. Rejected in favor of Execution-first
ownership with a bounded local correlation fallback.

## Consequences

### Positive

- Concise summaries can link to durable details without contaminating portable
  Project intent or Repository content.
- Execution remains the primary attempt concept; no parallel domain entity is added.
- Evidence can reference stable, digestible artifacts without treating all
  diagnostics as proof.
- Retention and cleanup preserve live workflow, recovery, review, and Evidence needs.
- Central ownership prevents Runtime/Provider adapters from inventing incompatible
  storage and cleanup semantics.

### Negative / trade-offs

- Axiom must maintain a machine-local artifact index/reference view or equivalent
  authoritative mechanism; its concrete representation remains future work.
- Reference-aware cleanup is more complex than time-based recursive deletion.
- Artifacts are not automatically portable or synchronized; cross-machine review
  needs an explicit later publication/export action.
- Exact size limits, retention durations, storage layout, and quota behavior remain
  Plan decisions and must be tested rather than inferred from this ADR.

## Revisit when

- Execution becomes available before every artifact-producing operation;
- cross-machine Evidence exchange requires a specified portable publication bundle;
- retention law, privacy, or compliance requirements impose new classes or deletion
  guarantees;
- measured volume or lookup cost requires a different local storage architecture;
- a concrete workflow proves that local correlation is insufficient and justifies
  another domain concept.

Any new domain entity, automatic cleanup policy, portable publication contract,
or retention guarantee requires explicit human review and compatible migration.
