# ADR-0023 — Local Runtime configuration conversational surface

## Status

**Proposed, 2026-10-11**, for [Issue #306](https://github.com/rgomids/axiom/issues/306)
in [Draft PR #310](https://github.com/rgomids/axiom/pull/310).
Requires independent exact-revision acceptance before implementation. No current
skill inventory or accepted contract is changed by this proposal.

## Context

[ADR-0022](0022-dedicated-workflow-conversational-surface.md) separates Project
permission, machine-local Runtime configuration and Workflow requirements.
Existing `axiom-workflow` configuration readiness is read-only; `axiom-project`
does not own host configuration. #306 requires supported conversational local
authoring without manual JSON/state manipulation, using one canonical store.

The historical Specification 004 two-skill consolidation was subsequently
extended by accepted #303 Workflow configuration ownership. A fourth active
skill requires an explicit inventory decision, not an incidental catalog edit.

## Proposed decision

Add thin `axiom-runtime` for machine-local Runtime bindings and named Model
Profiles. Canonical application/CLI operations own validation, selection,
preview/apply, exact revision, authorization, state and diagnostics. The skill
owns only intent collection, safe draft preparation and readable confirmation.
Both Codex and Claude discover the same embedded source/catalog contract.

Project chooses portable permission/active Workflow; Workflow defines Stage/Agent
requirements; Runtime configuration binds local models; Work Item requests and
controls execution. `axiom-runtime` cannot select Workflow revisions, edit Project
policy or run Work Items. Existing read-only Workflow readiness may refer users
to it; no authoring forwarding alias or new business registry is created.

The proposed [Spec 002 addition](../specifications/002-lingo-project-initialization/amendment-306-local-runtime-authoring.md#proposed-reconciliation-of-related-accepted-specifications)
defines the bounded active-inventory extension. On acceptance, reconcile the
current inventory references explicitly while preserving historical consolidation
records and receipt hashes. Keep retirements of old generic wrapper skills.

## Alternatives and trade-offs

| Option | Benefit | Cost / compatibility |
| --- | --- | --- |
| **Dedicated thin local skill (recommended)** | Clear machine-local ownership; supported conversational journey | Fourth installed skill, inventory/history/upgrade/parity checks; separate acceptance required |
| Expand `axiom-workflow` | Smaller inventory | Changes configuration boundary: Workflow would configure host state; accepted ownership amendment required |
| Expand `axiom-project` | Familiar entrypoint | Project-independent configuration becomes coupled to Project lifecycle; misleading scope of local effects |
| CLI only | Smallest implementation surface | Does not satisfy #306 conversational guidance acceptance |

No new server, MCP, workflow engine, secret backend or Runtime plugin framework
is introduced. Model IDs/effort tokens remain configuration and observed facts,
never hardcoded vendor quality tiers. This ADR approves no credential mechanism.

## Consequences and reversibility

Each intent has a clear owner and uses the canonical local authoring boundary.
Fresh install, upgrade and reinstall must prove both Runtime inventories,
catalog/help alignment, no conflicting active artifacts and preserved historical
receipts. Adding a discoverable skill raises installation/migration maintenance
cost; later removal requires owned retirement and another explicit decision.

Local wire format and portable Project contracts remain unchanged. The Plan's
credential/removal/effort decisions and mandatory security acceptance remain
separate gates. Implementation, Tasks, merge, release and acceptance of future
live scenarios are not authorized by this proposed ADR.

## Revisit when

Evidence shows users cannot understand local versus Project effects; a reviewed
credential backend is added; the active skill inventory changes; or supported
Runtimes require a materially different configuration/authentication contract.
