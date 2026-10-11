# ADR-0024 — Model Profile owns effort with versioned execution bindings

## Status

Accepted on 2026-10-11 by [explicit maintainer acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6109183466), fixed to commit `823bce3141bae5e90951c5239181c46d88cfbfe9`, blob `c9753a515ef43d2a84bae2156b1c6f8eac170dd1`. The exact historical artifact is preserved in Git; [acceptance receipt](../specifications/002-lingo-project-initialization/evidence-306-acceptance-tasks.md) records the six independent decisions. Acceptance establishes future architecture, not delivered code, installed inventory or native proof. Tasks preparation is authorized; implementation/merge/release remain separately gated.

## Context

Local ModelProfile format 1 has no effort. Workflow schema 1 requires effort per
Agent, which the compiler copies into immutable child controls. Accepted Spec 007
and ADR-0022 describe that ownership. D3 explicitly changes the owner to the local
Model Profile: Workflow references a Profile and must never override its effort.
This cannot be delivered by adding an optional field or deleting old Agent data.

## Decision

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

## Compatibility and evolution scope

The [accepted Spec 007 amendment](../specifications/007-configurable-workflows/amendment-306-profile-effort-onboarding.md)
defines the version matrix, migration, default template and acceptance criteria.
New readers retain local v1, Workflow v1 and supported old Execution/graph bytes
for inspection. Operator chooses effort during local v1→v2 migration; Workflow
v1→v2 publishes a new immutable revision with explicit Profile mapping. New
v2 admission refuses unmigrated ownership; existing Executions are never rebound.
New frozen bindings use accepted future configured Execution format 3 and graph/proposal
format 2; a legacy record without a proven v2 snapshot cannot gain one by resume.

No universal old-reader downgrade is promised. Incompatible readers fail safely;
FR-026 windows/forward paths and append-only compatibility corpus are required.
Monotonic local generation survives migration and retirement; recovery and rollback
require fresh exact authority and never reset it.

Only ADR-0022's assignment of effort to Stage/Agent and the corresponding Spec
007 effort/schema/Stage Plan clauses are partially superseded for the new versioned
contracts. Their historical text remains visible; v1 records retain original meaning.
ADR-0009 graph ownership and ADR-0020 immutable Workflow binding remain accepted.
No receipt, fixture, schema or historical bytes are rewritten by these annotations.

## Supersedes

- [ADR-0022 configuration scope](0022-dedicated-workflow-conversational-surface.md#configuration-scope-and-readiness-hd-006-direction): only Stage/Agent-owned configurable effort; all other boundaries remain in force.
- [Spec 007](../specifications/007-configurable-workflows/spec.md): only configurable Agent effort and Workflow-owned Stage Plan effort for future schema-2/versioned admissions, through its accepted #306 amendment. Legacy v1 meanings remain intact.

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
in the accepted Plan or a separately authorized Work Item before dependent implementation. Tasks may document that pending bundle. This ADR
creates neither. Adding versions increases compatibility maintenance and makes
upgrade reversible only through supported explicit paths; narrowing windows later
needs another accepted decision.

## Revisit when

A supported Runtime lacks safe no-override semantics, a reviewed proof source
changes its model/mode contract, or evidence requires a different legacy continuation
path. None permits silently moving effort back to Workflow configuration.
