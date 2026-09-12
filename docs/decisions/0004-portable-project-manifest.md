# ADR-0004 — Portable Project Manifest

## Status

Accepted on 2026-09-12, recording the explicit human decision during review of
[PR #4](https://github.com/rgomids/axiom/pull/4), the
[Specification 002 Plan](../specifications/002-lingo-project-initialization/plan.md).
Acceptance concerns this architectural choice; final Plan approval remains pending.
Tasks and Implementation are not authorized by this ADR.

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
repository sharing and synchronization remain open.

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

## Consequences

### Positive

- Project identity and intent survive copying and installation on another machine.
- Version-controlled review can inspect shared changes without local metadata.
- Lingo consumes an Axiom contract independent of Runtime and Provider products.
- Compatibility failures can be deterministic, preserving existing data.

### Negative / trade-offs

- Axiom owns schema documentation, validation and explicit evolution costs.
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
