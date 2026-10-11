# ADR-0024 — Model Profile owns effort with versioned execution bindings

## Status

**Proposed, 2026-10-11.** Reconciles the maintainer's
[D3 direction](https://github.com/rgomids/axiom/issues/306#issuecomment-6108025323)
in [Draft PR #310](https://github.com/rgomids/axiom/pull/310).
Direction is confirmed; this exact architecture/migration artifact remains
unaccepted. No Tasks, schema edits, implementation or vendor execution authorized.

## Context

Local ModelProfile format 1 has no effort. Workflow schema 1 requires effort per
Agent, which the compiler copies into immutable child controls. Accepted Spec 007
and ADR-0022 describe that ownership. D3 explicitly changes the owner to the local
Model Profile: Workflow references a Profile and must never override its effort.
This cannot be delivered by adding an optional field or deleting old Agent data.

## Proposed decision

Introduce required effort in local configuration format 2 and Workflow definition
schema 2 that forbids effort overrides. Resolve Runtime/model/Profile first,
derive effort only from that Profile, and bind it with exact configuration and
capability evidence into versioned preview/Stage Plan and immutable single/graph
Execution records. Preserve complexity as task metadata, distinct from effort.

Profile may explicitly choose an opaque adapter token or Runtime default
(no Axiom override; not a universal vendor effort level). Configuration is intent,
not proof; authoritative support gates use, not storage. Exact Runtime/model/executable/version/mode proof and fresh subscription
auth remain required before effects. Unknown support blocks; no alternate Profile
or effort is selected to make a run pass. Current observers do not supply G-3 proof.

Canonical store/resolver/graph remain sole authorities. Child controls are a frozen
projection of the Profile, not editable Workflow configuration. Historical effort
requests remain historical facts, never interpreted as current Profile defaults.

## Compatibility and proposed evolution scope

The [proposed Spec 007 amendment](../specifications/007-configurable-workflows/amendment-306-profile-effort-onboarding.md)
defines the version matrix, migration, default template and acceptance criteria.
New readers retain local v1, Workflow v1 and supported old Execution/graph bytes
for inspection. Operator chooses effort during local v1→v2 migration; Workflow
v1→v2 publishes a new immutable revision with explicit Profile mapping. New
v2 admission refuses unmigrated ownership; existing Executions are never rebound.
New frozen bindings use proposed configured Execution format 3 and graph/proposal
format 2; a legacy record without a proven v2 snapshot cannot gain one by resume.

No universal old-reader downgrade is promised. Incompatible readers fail safely;
FR-026 windows/forward paths and append-only compatibility corpus are required.
Monotonic local generation survives migration and retirement; recovery and rollback
require fresh exact authority and never reset it.

After acceptance, annotate only ADR-0022's configuration-scope sentence assigning
effort to Stage/Agent, and the corresponding accepted Spec 007 effort/schema and
Stage Plan clauses. ADR-0022 retains Accepted and all other boundaries. Add the
ADR-0018 reciprocal supersession metadata/index note **after** exact acceptance,
not in this Proposed revision. ADR-0009 graph ownership and ADR-0020 immutable
Workflow binding remain accepted; their codecs/projections evolve explicitly.
No historical receipt, r1 fixture, schema or accepted ADR is rewritten here.

## Alternatives and trade-offs

| Option | Benefit | Cost / reason |
| --- | --- | --- |
| **Versioned sole Profile owner (recommended)** | Meets D3; auditable migration; exact snapshots | Multiple codec/admission/projection families and explicit user migration |
| Optional effort added to local v1; keep Agent override | Small diff | Two owners and silent precedence contradict D3; rejects valid closed v1 semantics |
| Delete effort from Workflow v1 and reuse current snapshot | Less compatibility work | Changes historical meaning/digests and immutable records; rejected |
| Convert historical requests to Profile defaults automatically | Fast upgrade | Invents operator intent; conflicts across Agents/Projects and vendors; rejected |

## Consequences

Same model/different effort becomes two named local Profiles. Templates remain
vendor-neutral guidance; default-sdd r2 becomes ready only after explicit local
setup, Project permission and deterministic Profile choices. Historic r1 remains
addressable. G-3 and real onboarding/dispatch proof stay separate; documenting
interfaces does not implement capabilities or accept MVP operation.

The cross-cutting Workflow/compiler/Execution migration needs explicit ownership
in the accepted Plan or a separately authorized Work Item before Tasks. This ADR
creates neither. Adding versions increases compatibility maintenance and makes
upgrade reversible only through supported explicit paths; narrowing windows later
needs another accepted decision.

## Revisit when

A supported Runtime lacks safe no-override semantics, a reviewed proof source
changes its model/mode contract, or evidence requires a different legacy continuation
path. None permits silently moving effort back to Workflow configuration.
