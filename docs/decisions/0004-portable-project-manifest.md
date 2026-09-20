# ADR-0004 — Portable Project Manifest

## Status

Accepted on 2026-09-12, recording the explicit human decision during review of
[PR #4](https://github.com/rgomids/axiom/pull/4), the
[Specification 002 Plan](../specifications/002-lingo-project-initialization/plan.md).
Acceptance concerns this architectural choice. This ADR does not itself authorize
Tasks or Implementation. Current implementation lifecycle belongs to the
[Specification index](../specifications/README.md#002--lingo-project-initialization).

## Historical decision record

The dated entries below preserve the approval sequence and the gates that existed
at each review. Statements about pending Plan approval or implementation authority
are historical snapshots, not current lifecycle status.

### Subsequent reconciliation — 2026-09-12

Later human review of the same PR explicitly approved id/slug/name, the portable
working copy versus local installation state, incremental mutation and optional Git
backing/sync authority. This extends the accepted decision below under that explicit
human authority. That review did not rewrite Specification 002's original approval
on 2026-09-11 or approve the reconciled Plan. [H1–H8](../specifications/002-lingo-project-initialization/clarifications.md#subsequent-human-decisions--2026-09-12)
record exactly which earlier contracts changed. Previous ADR wording remains in Git
history; ADR-0001–0003 remain Accepted and unchanged.

### Update-contract confirmation — 2026-09-14

The [partial-input human decision](https://github.com/rgomids/axiom/pull/4#issuecomment-5664182508)
confirms the previous Project model and refines incremental update: partial
command intent is permitted, but domain materialization and validation of complete
proposed state must precede persistence. [H9](../specifications/002-lingo-project-initialization/clarifications.md#latest-human-decision--2026-09-14)
records authority and traceability. Original acceptance and H1–H8 remain historical;
that confirmation did not approve the final Plan or authorize Tasks/Implementation.

### Logical atomicity confirmation — 2026-09-14

The [H10 human decision](https://github.com/rgomids/axiom/pull/4#issuecomment-5664710882)
approves the Project model and init/update contract with logical atomicity.
[H10](../specifications/002-lingo-project-initialization/clarifications.md#logical-atomicity-confirmation--2026-09-14)
records this refinement; H11 below resolves the later Git Authority boundary
review. At that review, the final human Plan gate was pending; no Tasks or
Implementation was authorized.

### Git Authority approval — 2026-09-14

The [latest human decision](https://github.com/rgomids/axiom/pull/4#issuecomment-5665679754),
[H11](../specifications/002-lingo-project-initialization/clarifications.md#git-authority-approval--2026-09-14),
approves the Git Authority boundary below and confirms Project mental model,
init/update and H10 logical atomicity. At that review, final human Plan review
was pending; no Tasks or Implementation was authorized. Earlier approvals remain preserved.

## Context

An Axiom Project must preserve shareable intent and identity across machines,
independently of checkout locations, installed Runtimes and credential bindings.
[ADR-0001](0001-project-is-not-repository.md) separates Project from Repository;
[ADR-0003](0003-lingo-as-axiom-local-control-plane.md) makes Lingo the local
executor of Axiom contracts. A machine-local record alone cannot carry the shared
Project definition, and unversioned configuration leaves compatibility implicit.

[Specification 002](../specifications/002-lingo-project-initialization/spec.md)
and its [clarifications](../specifications/002-lingo-project-initialization/clarifications.md)
selected one strict `axiom.yaml` for the initial slice. Subsequent human Plan review
recognized the versioned portable manifest as a durable architectural contract.
This ADR records that later decision without rewriting Specification approval history.

## Decision

Axiom has an explicitly versioned **Portable Project Manifest** representing the
Project's shareable, version-controlled intent, consumed by Lingo and portable
between machines. It carries Project identity and can associate multiple independent
Repositories without requiring common providers, sibling paths or Git submodules.

The manifest may declare Runtime, Providers, Integrations, Capabilities, Model
Profiles, Business Context, Policies and logical credential references. Those
concepts remain separate: Project != Repository, Axiom != Lingo, Runtime != Model,
Provider != Transport and Integration != MCP. Their availability in the contract
does not make them mandatory or imply live connectivity or operational readiness.

Portable configuration is distinct from local state. Actual credentials, absolute
machine paths, executable locations, process identifiers, caches and temporary
state do not belong in the manifest. Local installations reference Project identity
and bind local resources without redefining it or rewriting shared intent.

Three levels must remain distinct:

- **Durable decision:** a Portable Project Manifest with explicit versioning and
  explicit contract/version evolution for incompatible changes.
- **Current representation:** `axiom.yaml`, using `schemaVersion` for the portable
  contract. A future representation change requires explicit review of compatibility
  and evolution; it is not silent acceptance of another format.
- **Initial concrete contract:** Specification 002 and its Plan, especially
  [Portable manifest contract](../specifications/002-lingo-project-initialization/plan.md#2-portable-manifest-contract),
  provide the current schema-specific documentation. Detailed v1 fields, encodings,
  requiredness and validation belong there and to future schema documentation,
  rather than being duplicated or globally frozen in this ADR.

No implicit compatibility, automatic migration or conversion follows from this
decision. The initial slice rejects unsupported versions without migration.
Internal Lingo installation storage is encapsulated behind the Local Installation
Store; its version/layout is a Plan detail, not a second portable contract or a
new architectural decision here. Project aggregate ownership, Workspace hierarchy,
repository sharing and concrete synchronization protocols remain open; optional
Git backing and authority separation are decided below.

### Identity, working copy and incremental mutation

Project remains logical, distinct from Repository. Canonical immutable UUID v4
survives name/slug rename, update, export, sync and another machine installation.
Slug is mutable human/CLI/location ergonomics, unique within one installation;
name is mutable nonunique presentation. Slug never becomes canonical identity.
Concrete slug grammar and collision/rename acceptance belong to Specification 002.

The per-user logical layout is `projects/<slug>/` for the portable/shared working
copy and `state/projects/<id>/installation.json` for installation state. Conceptually
these live under `~/.axiom`; existing native platform state conventions still map
the state role. A home directory location does not make portable intent private
machine state. Working copies contain manifest, optional context/documents/policies
and explicitly supported artifacts; no associated Repository is an implicit home
for them. Local paths, credentials, caches, PIDs, observations, Execution data and
bindings never belong in portable content. Local records keep internal
`formatVersion`, separately from portable `schemaVersion`.

Project may start with minimal identity and evolve. Init is create/no-op/conflict;
explicit update may receive a partial patch/intent. Lingo loads current Project;
the domain materializes a complete proposed state and validates all invariants.
Application presents the required safe diff and enforces applicable human/system
authority, version and concurrency requirements before persisting only the complete
valid result with logical atomicity. Direct partial mutation of `axiom.yaml` and bypass of
domain/application validation are forbidden for every caller. Slug rename
preserves UUID and local continuity, protecting collisions and filesystem races.
AI may propose drafts; deterministic application/domain controls validate identity,
schema, paths, conflicts, authority, versioning and persistence. No AI direct-write
bypass. These are portable intent boundaries, not a new aggregate or storage engine.

Never expose a partially updated Project as valid. Before commit the previous
valid Project remains authoritative and failures preserve it; after successful
commit the complete new valid Project becomes authoritative. Post-commit failures
report actual committed state and must not falsely claim rollback. This applies
to manifest/document updates and slug rename with identity/local-state continuity.

The adapter chooses concrete persistence mechanisms. Multi-file transactions,
staging layouts, locks, syscalls and transaction engines are not approved by this
ADR. Chosen mechanisms must provide Linux/macOS implementation Evidence for
confinement, collision protection, concurrency, logical atomicity and recovery.

### Optional distribution and synchronization

Portable working copy may later use dedicated Git backing independent of Repository
associations. A `.git` in that working copy creates no Repository association.
Symlinks remain a filesystem safety concern, not the primary export/sync model.

```text
Project mutation
      ↓
optional local Git commit
      ↓
optional explicitly authorized remote sync/push
```

These are separate operations with separate authority and outcomes. Project
mutation implies no Git and succeeds independently of commit/push. Later Git
failure cannot undo or falsely report rollback of confirmed local Project mutation.
Remote sync/push is never implied by init, update, installation, validation or
local Git commit. AI may propose operations but has no implicit authority for
local or remote Git effects. Machine-local state, credentials, bindings,
observations and other non-portable state never enter portable backing.

This Plan authorizes no implicit network mutation, hooks, credential-helper
changes, remote creation, branch rewrite, force push, merge/rebase or other Git
effects. Only the authority boundary is approved; Git execution is deferred.

Concrete Git backing schema/metadata, branch strategy, merge/rebase, conflict
resolution, sync protocol, transport, credential helpers, hooks, metadata
preservation, dirty-tree handling, recovery protocol, autoPush implementation and
remote-authority representation remain future design/implementation obligations.
None is selected by this boundary approval; appropriate Evidence and authorized
scope are required before delivery. Any autoPush/equivalent automation requires
explicit authority and separate human-approved design.

## Alternatives considered

### Machine-local configuration as the Project definition

Reduces initial serialization work but couples identity and intent to one machine.
Sharing requires reconstructing configuration and risks exposing local metadata;
second-machine installation becomes harder to operate and verify.

### Portable but unversioned configuration

Simplifies the initial file but makes compatibility implicit. Readers cannot safely
distinguish changed contracts; later evolution needs heuristics or lossy rewrites.

### Explicitly versioned Portable Project Manifest

Separates shared intent from local bindings and makes compatibility inspectable.
Adds strict validation and version-evolution responsibilities, with lower runtime
and machine coupling. This is the human-approved choice.

### Placement and evolution alternatives assessed in the later review

| Alternative | Trade-off and decision |
|---|---|
| Configuration in an associated Repository | Familiar local editing, but couples Project location and permissions to one delivery boundary; rejected under Project != Repository |
| One private local tree containing intent and bindings | Simpler writes, but sharing risks metadata/secrets and weakens portability; rejected |
| Shared working copy plus separate ID-addressed state | Adds discovery, slug collision/rename and cross-root recovery work; preserves identity, portability and least privilege; human-approved |
| Immutable creation only / manually bypass application to evolve | Smaller initial use case, but cannot validate/authorize legitimate incremental changes coherently; replaced by explicit deterministic update |
| Required Git, implicit push or primary symlink sync | Git adds operational/dependency cost; implicit push broadens authority; symlink ties distribution to local topology. Rejected as defaults |
| Optional dedicated Git backing with explicit sync | Adds future conflict/recovery design, but separates delivery associations and remote authority; human-approved boundary only |

ADR assessment: all four candidate decisions refine the same portable-intent/local
installation boundary already owned here. Extending ADR-0004 avoids splitting one
source-of-truth contract across competing ADRs. No separate engine, aggregate or
remote topology ADR was justified at that review. ADR-0005 was later assigned to
the unrelated bounded local filesystem threat model. Later Git implementation may reveal
an independent durable decision requiring its own evidence and human approval.

## Consequences

### Positive

- Project identity and intent survive copying and installation on another machine.
- Version-controlled review can inspect shared changes without local metadata.
- Lingo consumes an Axiom contract independent of Runtime and Provider products.
- Compatibility failures can be deterministic, preserving existing data.

### Negative / trade-offs

- Axiom owns schema documentation, validation and explicit evolution costs.
- Slug namespace, safe location changes, logical Project atomicity and local-state
  recovery add filesystem/concurrency complexity. ID and published slug semantics
  become harder to change after distribution; no automatic migration is promised.
- Optional Git reduces mandatory coupling but defers sync conflict handling and
  metadata design; remote publication must retain its independent authority gate.
- Readers must distinguish unsupported versions and portable validity from local
  installation and readiness; a valid manifest alone cannot make a workflow run.
- Published fields and version semantics become harder to change after adoption;
  incompatible changes require deliberate contract/version evolution.
- Local bindings must be re-established or revalidated on each machine.

## Revisit when

- Validated collaboration needs require a different portable representation or
  synchronization model beyond the current manifest boundary.
- Schema evolution cannot preserve explicit compatibility and safe failure.
- A specified workflow demonstrates that the portable/local split cannot represent
  required intent without leaking machine state or secrets.

Reconsideration requires evidence and human review, not automatic migration or
an implementation-stage shortcut. ADR-0001–0003 remain unchanged; Spec-Kit remains
strategic upstream research under [ADR-0002](0002-axiom-speckit-relationship.md).
