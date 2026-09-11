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
requires only Project ID/name, >=1 Repository association and schemaVersion;
Runtime, Providers, Integrations, profiles and Business Context may be absent or
unconfigured. Provider requirements need a concrete workflow/Capability.
Provider choices remain independent. Work Items are not provider records, a
Project is not a Repository, and runtime/model availability must be resolved
through capability discovery rather than a hardcoded domain catalog.

## Candidate first executable slice

`lingo project init` is the candidate first vertical slice:

```text
Project ID + name + schemaVersion
-> At least one Repository association
-> Optional Runtime / Providers / Integrations / profiles / context
-> Validate
-> Persist
```

The minimum slice should prove portable Project definition, safe local state,
runtime-aware configuration, and deterministic validation without implementing
the full orchestration platform.

No implementation begins before a Specification defines behavior, non-goals,
security boundaries, failure handling, acceptance evidence, and migration
impact. [ADR-0003](../decisions/0003-lingo-as-axiom-local-control-plane.md)
accepts Lingo as the architectural local control-plane direction; it does not
authorize implementation by itself.

The approved [Specification 002 — Lingo Project Initialization](../specifications/002-lingo-project-initialization/spec.md)
combines the minimum configuration behavior behind blocks 1–4 and 7, including
portable creation, reopening and local installation. Its
[clarifications](../specifications/002-lingo-project-initialization/clarifications.md)
record Q1–Q6 as resolved by human review: UUID v4, one strict `axiom.yaml`,
OS-native local state for Linux/macOS with root override, presence-only Runtime
observations, minimal portable validity with explicit installation gaps, and
conservative Repository matching. Specification 002 is Approved by human review
on 2026-09-11. The first Lingo vertical slice has an approved Specification and
is ready for planning; it is not implemented. No blocking clarification remains.
Next step: merge PR #3, then start Plan in a new change. Implementation remains
gated by an approved Plan; this PR creates no Plan, Tasks or implementation.
