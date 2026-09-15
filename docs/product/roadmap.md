# Product Roadmap

## Status

Architectural sequencing for discovery and specification. This is not a commit
plan, implementation authorization, or promise that each block becomes a
separate pull request.

## Next capability blocks

1. **Lingo as Axiom Control Plane** — specify the accepted Axiom/Lingo boundary,
   command surface, lifecycle responsibility, and acceptance evidence before implementation.
2. **Project Manifest and Portable/Local Configuration** — specify Project
   identity, shareable configuration, machine-local state, credential
   references, validation, and installation behavior.
3. **Runtime and Model Abstraction** — specify runtime discovery, Agent and
   Model Profiles, user configuration, capability reporting, and portability.
4. **Capability and Integration Model** — define provider-neutral capability
   vocabulary, Integration scope, Transport boundaries, negotiation, and
   missing-capability behavior.
5. **Execution Graph and Agent Orchestration** — specify Agent Planner,
   Orchestrator, parent/child Executions, dependencies, Evidence, approvals,
   failure, and reconciliation.
6. **Thin Runtime Skills and Adapters** — define runtime entrypoints and adapter
   contracts without duplicating Axiom workflows.
7. **Project Init Wizard** — specify guided Project creation after the required
   configuration and capability contracts are understood.

Ordering expresses dependency pressure, not a frozen delivery sequence.
Evidence may combine, reorder, split, or defer blocks.

## Future Project Wizard

A future `lingo project init` flow is expected to guide:

1. Project;
2. agent Runtime;
3. Work Item Provider;
4. source-control Provider;
5. one or more Repository associations;
6. documentation Provider;
7. Integrations;
8. Model Profiles;
9. Business Context;
10. validation;
11. Project creation.

These are possible wizard topics, not mandatory declarations. Specification 002
requires only Project ID/slug/name and schemaVersion after the 2026-09-12
human revision; Repository associations may be absent/empty;
Runtime, Providers, Integrations, profiles and Business Context may be absent or
unconfigured. Provider requirements need a concrete workflow/Capability.
Provider choices remain independent. Work Items are not provider records, a
Project is not a Repository, and runtime/model availability must be resolved
through capability discovery rather than a hardcoded domain catalog.

## Candidate first executable slice

`lingo project init` is the candidate first vertical slice:

```text
Project ID + slug + name + schemaVersion
-> Portable working copy under projects/<slug>
-> Optional Repository associations
-> Optional Runtime / Providers / Integrations / profiles / context
-> Validate
-> Persist atomically
-> Explicit update with diff/authority/revision, preserving ID
```

The minimum slice should prove portable Project definition, safe local state,
runtime-aware configuration, and deterministic validation without implementing
the full orchestration platform.

No implementation begins before a Specification defines behavior, non-goals,
security boundaries, failure handling, acceptance evidence, and migration
impact. [ADR-0003](../decisions/0003-lingo-as-axiom-local-control-plane.md)
accepts Lingo as the architectural local control-plane direction; it does not
authorize implementation by itself.

[Specification 002 — Lingo Project Initialization](../specifications/002-lingo-project-initialization/spec.md)
combines blocks 1–4 and 7: minimal creation, explicit incremental update, reopening
and local installation. Original human approval was 2026-09-11. Subsequent PR #4
Plan review on 2026-09-12 reopened location/minimum/update contracts through
[H1–H8](../specifications/002-lingo-project-initialization/clarifications.md#subsequent-human-decisions--2026-09-12),
preserving original Q1–Q6 history. [H9, confirmed 2026-09-14](../specifications/002-lingo-project-initialization/clarifications.md#latest-human-decision--2026-09-14), permits partial update intent with complete domain materialization/validation before persistence. H10 preserves logical atomicity without choosing persistence mechanisms; [H11](../specifications/002-lingo-project-initialization/clarifications.md#git-authority-approval--2026-09-14) approves Git Authority. Current [Plan](../specifications/002-lingo-project-initialization/plan.md)
is reconciled against that revised Specification; T01 delivers the pure Project domain; T02 delivers application boundary contracts.

Extended [ADR-0004](../decisions/0004-portable-project-manifest.md) records canonical
ID versus slug/name, portable working copy distinct from ID-addressed local state,
incremental mutation and optional dedicated Git backing. Git commit/export/sync
execution remains deferred; remote authority is explicit and never an init/update
side effect. Backing Git is not automatically a Repository association. Internal
`formatVersion` stays separate from portable `schemaVersion`; declaration forms
retain their intent distinctions.

PR #4 approved and merged on 2026-09-14. **Plan: Approved.**
[Tasks](../specifications/002-lingo-project-initialization/tasks.md): **Approved** after PR #5 merge.
**Implementation: Authorized (T02 only). T01: Accepted / merged**, PR #6.
[T02 Evidence](../specifications/002-lingo-project-initialization/evidence-t02.md):
**Ready for human implementation review. T03–T21: Not started.**
