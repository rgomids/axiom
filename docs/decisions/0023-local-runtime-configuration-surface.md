# ADR-0023 — Local Runtime configuration conversational surface

## Status

Accepted on 2026-10-11 by [explicit maintainer acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6109183466), fixed to commit `823bce3141bae5e90951c5239181c46d88cfbfe9`, blob `f39a75b1d227016bbd19b096bfae326ad92049f1`. The exact historical artifact is preserved in Git; [acceptance receipt](../specifications/002-lingo-project-initialization/evidence-306-acceptance-tasks.md) records the six independent decisions. Acceptance establishes future architecture, not delivered code, installed inventory or native proof. Tasks preparation is authorized; implementation/merge/release remain separately gated.

## Context

[ADR-0022](0022-dedicated-workflow-conversational-surface.md) separates Project
permission, machine-local Runtime configuration and Workflow requirements.
Existing `axiom-workflow` configuration readiness is read-only; `axiom-project`
does not own host configuration. #306 requires supported conversational local
authoring without manual JSON/state manipulation, using one canonical store.

The historical Specification 004 two-skill consolidation was subsequently
extended by accepted #303 Workflow configuration ownership. A fourth active
skill requires an explicit inventory decision, not an incidental catalog edit.

## Decision

Add thin `axiom-runtime` for machine-local Runtime bindings and named Model
Profiles. Canonical application/CLI operations own validation, selection,
preview/apply, exact revision, authorization, state and diagnostics. The skill
owns only intent collection, safe draft preparation and readable confirmation.
It lists safe known options, explains current blockers and guides first configuration
without requiring hand-authored JSON. It cannot authenticate vendors or infer
model/effort support from installation.
Both Codex and Claude discover the same embedded source/catalog contract.

Project chooses portable permission/active Workflow; Workflow defines Stage/Agent
requirements and Profile references; Runtime configuration binds local models and
sole Profile-owned effort; Work Item requests and
controls execution. `axiom-runtime` cannot select Workflow revisions, edit Project
policy or run Work Items. Existing read-only Workflow readiness may refer users
to it; no authoring forwarding alias or new business registry is created.

The accepted [Spec 002 addition](../specifications/002-lingo-project-initialization/amendment-306-local-runtime-authoring.md#reconciliation-of-related-accepted-specifications)
defines the accepted bounded active-inventory extension. Reconcile the
future inventory references explicitly while preserving historical consolidation
records and receipt hashes. Keep retirements of old generic wrapper skills.

## Alternatives and trade-offs

| Option | Benefit | Cost / compatibility |
| --- | --- | --- |
| **Dedicated thin local skill (recommended)** | Clear machine-local ownership; supported conversational journey | Fourth future installed skill; inventory/history/upgrade/parity proof required |
| Expand `axiom-workflow` | Smaller inventory | Changes configuration boundary: Workflow would configure host state; accepted ownership amendment required |
| Expand `axiom-project` | Familiar entrypoint | Project-independent configuration becomes coupled to Project lifecycle; misleading scope of local effects |
| CLI only | Smallest implementation surface | Does not satisfy #306 conversational guidance acceptance |

Distribute optional `fast`, `balanced`, `deep` guidance without fixed models,
universal effort levels or vendor rankings. One valid Profile per detected Runtime
is enough structurally; preserve existing config, never create an absent binding.
CLI install stays noninteractive; first use guides via this skill. Every local
mutation uses exact confirmation; Project policy/Workflow publication/active
selection and Work Item start remain separate approvals. Known model/effort options
require authoritative provenance; otherwise report unproven with consent handoff.
The [Spec 007 amendment](../specifications/007-configurable-workflows/amendment-306-profile-effort-onboarding.md)
separates distributed default-sdd template from user-confirmed derived config.

No new server, MCP, workflow engine, secret backend or Runtime plugin framework
is introduced. Model IDs/effort tokens remain configuration and observed facts,
never hardcoded vendor quality tiers. This ADR approves no credential mechanism.

## Consequences and reversibility

Each intent has a clear owner and uses the canonical local authoring boundary.
Fresh install, upgrade and reinstall must prove both Runtime inventories,
catalog/help alignment, no conflicting active artifacts and preserved historical
receipts. Adding a discoverable skill raises installation/migration maintenance
cost; later removal requires owned retirement and another explicit decision.

Portable Project contracts remain unchanged. Local v2 Profile effort and
versioned Workflow/Execution migration are accepted separately in ADR-0024;
this skill cannot implement that decision by changing guidance alone. D1–D5
contracts and independent design-security baseline are accepted. Exact Tasks
approval, implementation, merge, release and acceptance of future live scenarios
remain separate gates; this ADR supplies none of those authorities.

## Revisit when

Evidence shows users cannot understand local versus Project effects; a reviewed
credential backend is added; the active skill inventory changes; or supported
Runtimes require a materially different configuration/authentication contract.
