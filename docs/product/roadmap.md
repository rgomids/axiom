# Product Roadmap

## Status

Architectural sequencing for discovery and specification. This is not a commit
plan, implementation authorization, or promise that each block becomes a
separate pull request.

## Next capability blocks

1. **Lingo as Axiom Control Plane** — refine the proposed Axiom/Lingo boundary,
   command surface, lifecycle responsibility, and acceptance evidence.
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

Provider choices remain independent. Work Items are not provider records, a
Project is not a Repository, and runtime/model availability must be resolved
through capability discovery rather than a hardcoded domain catalog.

## Candidate first executable slice

`lingo project init` is the candidate first vertical slice:

```text
Project
-> Runtime
-> Repository
-> Work Item Provider
-> Documentation Provider
-> Model Profile
-> Validate
-> Persist
```

The minimum slice should prove portable Project definition, safe local state,
runtime-aware configuration, and deterministic validation without implementing
the full orchestration platform.

No implementation begins before a Specification defines behavior, non-goals,
security boundaries, failure handling, acceptance evidence, and migration
impact. [ADR-0003](../decisions/0003-lingo-as-axiom-local-control-plane.md)
remains Proposed until human review.
