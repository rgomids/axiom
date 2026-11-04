# Issue #306 — Profile-owned effort and default Workflow onboarding

> **AXM-42 prospective reconciliation — 2026-10-30.**
> [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md#supersedes)
> records the [human material decision](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5720318529)
> approving [Spec 015 invocation ownership](../015-minimum-viable-operational-workflow/spec.md#invocation-ownership-and-precedence-proposed-f5).
> For new attempts only, PE-01–PE-05/PE-05a and AC-PE1/AC-PE3 change where they
> require sole Profile effort, forbid direct overrides, require distinct
> Profiles for different efforts or project only Profile provenance; PE-04 and
> the version matrix no longer oblige operational old-format readers after the
> breaking cutover. Truthful identity, safe source inspection, immutable
> provenance and no fallback remain. Historical text below is preserved.

> **AXM-36 prospective reconciliation — 2026-10-19.**
> [ADR-0032](../../decisions/0032-runtime-admission-without-certification.md#supersedes)
> was [accepted at `b4baf106`](https://github.com/rgomids/axiom/pull/334#issuecomment-6278024276).
> Only PE-03/04/05/05a/06 proof, proof-reference and exact-version eligibility obligations
> change within [RA-6](amendment-axm36-runtime-admission.md#ra-6--verified-coupling-and-prospective-reconciliation).
> Historical text and Evidence below are preserved. Profile ownership, safe
> source inspection, identity, policy, authority and supervisor contracts remain.
> The [Proposed Plan](plan-axm36-runtime-admission.md) specifies the bounded
> structural retirement and successor-format design; it grants no implementation.

## Status and authority

**Accepted, 2026-10-11**, by [explicit maintainer acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6109183466), at commit `823bce3141bae5e90951c5239181c46d88cfbfe9`, blob `4bc5b65830fb9ec885bdd53c2484f53c13a02b86`. [Acceptance receipt](../002-lingo-project-initialization/evidence-306-acceptance-tasks.md) preserves exact authority and history. Accepted D3 modifies future contracts through versioned migration; Workflow schema 1 and old Executions retain their meaning. Tasks preparation is authorized; this revision reconciles status/backlinks only, with no implementation/schema/fixture/vendor change.

Companions: [Plan](../002-lingo-project-initialization/plan-306-local-runtime-profiles.md), [Spec 002 amendment](../002-lingo-project-initialization/amendment-306-local-runtime-authoring.md), [Security](../002-lingo-project-initialization/security-review-306-local-runtime-profiles.md), [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md).

## 1. Verified current contract and bounded delta

Current Workflow `schemaVersion: 1` requires `Agent.effort` in the closed schema.
`workflowdefinition.Agent`, `validation.go` and `workflowcompiler.resolveAgent`
implement that contract. Local `runtimeprofile.ModelProfile` format 1 has no
effort. `compiledUnit` derives graph controls from Agent effort; Stage Plan v1
reports Workflow `requestedEffort`. `workflow.bindingDocument` accepts only
Workflow schema 1 in configured Execution format 2. These are historical facts,
not implementation defects under their accepted contracts.

Both default-sdd-r1 copies have ten agents, role `engineer`, complexity `medium`,
constraints `[codex, claude]`, `profileRef: policy-default`, and
`effort: {mode: runtime-default, value: default}`. `resolveAgent` tries constrained
Runtimes separately and requires exactly one successful result. `policy-default`
becomes an unspecified Profile ID, resolved by portable Project preferences and
the canonical local intersection. Two viable results are a blocker; sorted
Runtime iteration is not selection priority.

This accepted amendment changes only future effort ownership/versioned bindings and the
guided path to a ready default Workflow. Project allowlist/permissions and
immutable binding ownership remain Spec 002 / ADR-0020; no new scheduler,
Provider authority, Project policy editor or automatic workflow selection.

## 2. Accepted contract: one effort owner

**PE-01.** Local configuration format 2 requires `ModelProfile.effort` as a closed
object: `{mode: explicit, value: <opaque adapter token>}` or
`{mode: runtime-default, value: default}`. It is explicitly chosen by the operator,
not inferred from Profile name, complexity, model or initiating Runtime.
`runtime-default` means the Profile requests no Axiom effort override; it does
not claim a concrete internal vendor level. Missing effort is unconfigured,
never silently default. Two globally unique Profile IDs can share Runtime/model
with different effort. Profile enabled eligibility remains Runtime allowlist
membership; no duplicate persisted effort source or Profile enabled flag.

**PE-02.** Workflow definition schema 2 retains role, complexity, Profile reference
and Runtime constraints, but **forbids Agent `effort`**, including null/empty
forms. New Workflow authoring, planned WorkUnit inputs and CLI options cannot
override it. Reserve `reasoning-effort-*` requirements for canonical derivation
from the selected Profile: reject them as user-authored Workflow capabilities
or alternate selectors. Local v2 authored capability lists likewise cannot
encode a second effort choice; legacy tokens remain historical v1 data only.

**PE-03.** The sole resolver selects an exact permitted Profile first, then derives
that Profile's effort requirement and validates the same candidate against proof.
If its effort fails, it blocks that choice; it does not search for a different
effort/Profile. More than one otherwise eligible candidate is ambiguous, even
when one effort would work. Profile choice uses an explicit reference, exact Project role/complexity preference,
or a unique permitted candidate disclosed in the reviewed preview and explicitly
confirmed. Multiplicity still blocks; no quality ranking or vendor translation.

**PE-04.** Accepted future v2 resolution projection carries a safe immutable
`resolvedProfile` tuple: `profileId`, `runtimeId`, `model`, `effort`,
`profileDigest`, `configurationRevision`, `configurationDigest`,
`executableDigest`, `runtimeVersion`, `invocationMode`, `observationRevision`,
`observationDigest`, `effortSourceDigest` and capability proof reference. Digests bind canonical values,
including effort; no raw auth output or paths. A versioned v2 use case/projection
distinguishes this tuple from historical choice/Stage Plan DTOs. Keep legacy v1
payload decoding and meaning; never relabel Workflow `requestedEffort` as local
effort. Stage Plan v2 reports `profileEffort` and effective override/no-override
semantics with `effortSource: local-model-profile`.

**PE-05.** Compiler v2 derives immutable controls/capability tokens solely from
that tuple; graph/one-agent dispatch compares controls with it. Controls are a
frozen execution projection, not another configurable owner. No inherited
argv/env/vendor setting may replace the approved effort, including a Profile's
explicit no-override choice. Native model-internal defaults remain native when that
mode is explicitly selected; no concrete internal effort level is asserted. Reuse
existing argument/environment guards, exact adapter encoding and fresh subscription
authentication boundary, then add the bounded effort-source check below.

**PE-05a — bounded external effort inspection.** At a separately consented preflight
or authorized dispatch boundary, inspect recognized effort selectors without editing
vendor settings. Accepted inspection sources mirror existing bounded configuration scopes:
Codex effective argv, environment and user/managed/project `config.toml` settings
(including `model_reasoning_effort` and selected named config sections); Claude
effective argv, `CLAUDE_CODE_EFFORT_LEVEL`, user/project/project-local/managed
settings and supported managed drop-ins. Source identities/roots are internal,
never public host paths. Each adapter must declare supported fields/source scopes
and precedence for its exact observed version/mode from authoritative evidence;
unknown configuration, unsupported precedence, malformed/unreadable sources or
ambiguous selected config sections block `effort_configuration_unproven`.
Current auth preflight does not provide this effort proof. Do not invent new
Claude settings keys or unsupported vendor behavior. For runtime-default, any
external explicit effort selector blocks `effort_override_conflict`; ordinary
native model defaults with no selector are permitted as delegated no-override
semantics. For explicit mode, admit only the canonical Profile-derived override;
other configured/inherited selectors cannot be merged or silently win. Report
source classes/key names and safe categories only; never raw configs/environment,
login files, account values or credential payloads. No status probe, repair,
setting removal or forced precedence is granted by inspection. Bind the safe
canonical effort-source projection (recognized source classes/key-presence and
compatibility facts, not raw values/paths/config files) as effortSourceDigest in
the reviewed tuple; re-evaluate it at dispatch. Unknown/unreadable sources cannot
be approved by hashing them. Explicit consent to a probe never auto-applies a
configuration or starts a model run.

**PE-06.** Explicit effort use needs authoritative evidence for exact Runtime,
model, executable digest, client version, noninteractive invocation mode and
token, with freshness bound into preview and checked before each attempt.
Proof must cover the selected model, not just Runtime-wide support. The current
observer proves executable/integration only; authentication status does not prove
model availability/effort. Model execution capabilities and no-override adapter
semantics still need their appropriate evidence when runtime-default is chosen.
Save may persist explicit intent with `unproven` diagnostics; it cannot report
operational readiness or dispatch it. G-3 proof acquisition remains separate,
requiring its own authority; this planning run performs no vendor operation.

## 3. Versioning and explicit migration strategy

Versions below are accepted future implementation contracts; no schema/fixture is changed
in this PR. Codec compatibility and monotonic record revision remain independent.

| Family | New contract | Supported historical handling / forward path |
| --- | --- | --- |
| Local active config | `formatVersion: 2`, required Profile effort; same canonical store/path | Read format 1 without rewriting. Mark effort unconfigured for future v2 use; explicit migration candidate chooses effort per Profile, increments revision once |
| Local retirement | `runtime-profile-tombstone`, format 1 metadata | Tombstone generation spans active formats 1/2; next creation always active v2 at prior + 1, never resets generation |
| Workflow definition | `schemaVersion: 2`; effort forbidden | Decode schema 1 exactly for inspection/history. Explicit migration publishes next immutable revision of the same Workflow ID; leaves source bytes/digest unchanged |
| Distributed default | `default-sdd` revision 2/schema 2 | Preserve revision 1 and digest as addressable historical built-in; never replace the bytes behind an existing builtin reference |
| Stage Plan / compiler result | Version 2 projection and effort provenance | Read v1 as Workflow-owned historical request; old review digests cannot approve a v2 result |
| Configured Execution | Accepted future format 3 / `configured-sequential-v2`, schema-2 Workflow snapshot | Existing format 1/2 inspection/history preserved; extend canonical codec/inventory without rewriting old records |
| Graph/proposal | Version 2 with resolvedProfile snapshot per node | Version 1 records stay readable and immutable; no invented provenance or automatic upgraded proposal |

Migration is **prepare → validate → preview → explicit confirmation → apply →
read-back**, separately for each owner. Local migration never infers effort from
legacy Agent requests or capability tokens. Inventory of legacy Agent effort
provides a safe comparison report: the user may choose/create separate Profiles
for each intended combination, preserving explicit intent through reviewed new
bindings. An incompatible replacement is an explicit changed behavior with a new
preview, not conversion success. Mixed v1/v2 Workflow fields are rejected.

Workflow migration materializes schema 2 from a specific immutable schema-1
revision and a user-confirmed per-Agent Profile mapping. It reports each removed
legacy effort request beside the selected Profile effort in its proposal, then
publishes a new revision. Local Profile creation and portable policy edits are
separate confirmations; Project model/key/Runtime matching remains exact.
Existing Project policy editing is #305. When it is unavailable or does not
allow a requested Profile, report `project_policy_update_required`; never patch
portable files from #306. Active Workflow selection is the separate Project
operation after new revision publication. No multi-owner transaction is implied.

The compatibility window for the first release implementing this contract reads
local configs 1/2, tombstone 1, Workflow definitions 1/2, and supported historical
Execution/graph families above through canonical readers/inventory. New v2
admission requires explicit effort and a schema-2 Workflow. Legacy definitions
remain readable/selectable for historical inspection, but new starts or new stage
compilation under Profile-owned semantics return `workflow_migration_required`.
They cannot ignore a legacy Agent override. Local v1 similarly returns
`profile_effort_unconfigured` for new v2 admission, not invalid/unsupported data.

> **Prospectively superseded for local configuration — 2026-10-14.** Maintainer
> decision C1 (13:30 America/Sao_Paulo), recorded as
> [AXM-22 clause PM-8](amendment-276-axm22-binding-identity.md#pm2-pre-stable-compatibility-and-canonical-digestv3-bld-001b3c-2a),
> supersedes for the pre-MVP delivery only the *Local active config* and
> *Local retirement* forward paths above and the reading of local configs 1/2 in
> the first-release compatibility window: before the first stable/LTS release
> there is no v1→v2 migration requirement, configuration format 3 is the only
> target canonical format and unsupported pre-stable local state requires
> explicit reconfiguration. The Workflow, distributed default, Stage Plan,
> configured Execution and graph rows, and Profile effort ownership, remain in
> force. The text above is preserved as history.

Existing Executions retain their schema-1 WorkflowBinding and every prior graph,
attempt and effect. Never rebind/recompile them from a migrated Project selection.
New dispatch/retry that cannot demonstrate a v2 immutable Profile/effort snapshot
is blocked with `legacy_execution_contract` and explicit reconciliation guidance.
Do not synthesize a snapshot from mutable local state. Safe inspection and
historical completion reading remain available. An authorized new/continuation
Execution, if supported by its existing lifecycle owner, creates a new identity;
#306 does not implement that lifecycle or automatically create a successor.

For a new Execution, Start freezes its selected schema-2 Workflow and initial
resolved Profile. Subsequent authorized stage admission freezes all actual child
resolvedProfile tuples with the Stage Plan/proposal in the same protected
publication. This may append new lifecycle records; it never edits existing
bindings. Local changes after a stage preview invalidate that preview. Changes
after a tuple is frozen cannot replace its effort; attempt checks block drift
until the lifecycle's explicit replan/reconciliation process grants new authority.

MVP promises a supported forward upgrade/read path, not universal downgrade.
Old readers fail safely on local v2, tombstones, Workflow schema 2 and new
Execution/graph formats. Downgrade is unsupported with those records; no automatic
down-conversion or deletion. Recovery restores exact prior bytes only for provably uncommitted publication,
or finalizes its provable commit. Rollback of a committed migration/retirement/
recreation publishes a new authorized generation, never resetting revision. Stable snapshots and
receipts are append-only; add new versioned corpus/recognition tests under FR-026.

## 4. Onboarding and default-sdd journey

**ON-01 — passive first use.** CLI installation stays noninteractive. Reuse the
deterministic executable detection logic, not `runtimebootstrap.Run` for read-only
inventory: `Run` can install Axiom integrations. Report separately installed,
integrated, authenticated/unobserved, model availability and effort support.
No vendor process or credential read during default inventory. On first
configuration-dependent use, missing/insufficient state points to `axiom-runtime`;
the conversation guides a draft, never silently starts a probe or writes state.
Valid existing configuration is preserved byte-for-byte until a requested,
confirmed migration/edit; inconsistent state triggers scoped recovery guidance.

**ON-02 — optional templates.** Distribute `fast`, `balanced`, `deep` as descriptive
local authoring templates, not configured Profiles, quality tiers or vendor
defaults. They contain no model ID, effort token or universal complexity-to-effort
mapping. User chooses any names and supported models/efforts. Globally unique IDs
may be `codex-fast`/`claude-fast`, but none is automatically instantiated.
One valid Profile per detected Runtime suffices structurally; three are optional.
This does not imply that one Profile satisfies every future task capability.
The user may configure an explicitly chosen installed subset; detected-but-unused
Runtimes remain unconfigured with truthful diagnostics, never forced into state.

**ON-03 — truthful options.** Skill lists actual supported installed Runtime
options and safe known capability/model/effort facts with provenance and freshness.
Only authoritative observations support recommendations. Missing facts result in
`model_availability_unproven`/`effort_support_unproven` and a consented observation
handoff, not a model invented from chat identity or vendor rankings. User may save
an explicit structurally valid candidate as unproven; ready requires separate
auth/model/effort evidence. Detected installation is a prerequisite for authoring a binding; the supported
operational scenario additionally requires prior vendor-managed CLI authentication.
Unobserved authentication does not force a probe or prevent saving intent: it blocks
operational readiness/dispatch, not structural local authorship. Axiom never
installs/authenticates vendors or handles tokens.

**ON-04 — separate approvals.** Prepare and confirm local Profile candidate, apply
and read-back through G-2. Then use `axiom-project` to create/read allowed policy
and `axiom-workflow` to prepare a derived Workflow revision. Each mutation has its
own exact preview. Project selects active revision independently. Work Item then
uses its existing planning/start flow; its authority is not inherited from setup.
No skill edits internal JSON or owns a parallel onboarding state store.

**ON-05 — default template is not selected configuration.** Accepted future default-sdd
revision 2 has no concrete vendor/model/effort choice, retains supported Runtime
constraints and `policy-default`, and requires the canonical intersection.
For one eligible Profile, show the exact resolution before confirmation; detection
alone never selects it. With multiple candidates, ask for explicit per-Agent
Profile references in a confirmed Project-owned copy, or a separately authorized
portable role/complexity preference. Do not invent local preferences as Project
intent. The new template may be offered, never auto-selected on an existing
Project or substituted behind an Execution's frozen revision.

| Installed state | Local setup and derived Workflow | Readiness outcome |
| --- | --- | --- |
| None | No Runtime binding; explain prerequisite | Installed Runtime missing; zero mutation |
| Codex only | At least one explicit Codex Profile; no Claude binding; confirmed refs/policy | Unique permitted observed candidate or precise auth/model/effort blocker |
| Claude only | At least one explicit Claude Profile; no Codex binding; confirmed refs/policy | Same owner/authority rules; no Codex default |
| Both | At least one explicit Profile per Runtime; explicit per-Agent assignment or Project preference | Unique assignment supports single/mixed Runtime plans; two unresolved candidates block |
| Existing valid state | Read and preserve; guide only missing requirements | Unchanged unless requested edit/migration; no replacement by templates |
| Existing legacy/unsafe/pending state | Safe report; explicitly reviewed migration/recovery | No auto-default, silent rewrite, fallback or unsupported success |

## 5. Scope ownership and acceptance

#306/G-2 owns local v2 Profile authoring/onboarding, configuration skill and safe
authoring/projection interfaces. It proposes the Workflow v2/immutable binding
delta because D3 makes it a prerequisite; implementation ownership remains
Pending Human Decision before dependent implementation. Tasks document the entire
bundle separately without granting it #306 implementation authority. Recommend a separately bounded technical
Work Item for cross-cutting Workflow/compiler/Execution-format migration if that
delta should not be included in #306. No Issue is created here. #305 owns existing
Project policy editing; G-3 owns authoritative proof acquisition; #276/#277 own
dispatch/delivery lifecycle; #278 owns actual R-2/R-3 acceptance. Publishing this
amendment unblocks none of them automatically.

| Criterion | Required objective Evidence in future implementation |
| --- | --- |
| AC-PE1 sole owner | Local v2 effort required; Workflow v2 effort/effort-capability/WorkUnit override inputs rejected; same model/different effort Profiles produce distinct digests and frozen controls |
| AC-PE2 preservation | v1 configs/Workflow/execution/graphs readable byte-for-byte; no automatic migration; explicit v2 conversion and new revision/identity; old approval digests refused |
| AC-PE3 snapshot and proof | Exact Profile/config/proof tuple bound for single and graph paths; edit/token/model/executable/version/mode drift and external Claude effort env/Codex effort config block before effects; declared/runtime-wide/auth-only proofs fail |
| AC-ON1 first-use matrix | None/Codex-only/Claude-only/both; deterministic absolute executable presence; no absent bindings; one/three/over-three Profiles; no required interactive CLI install |
| AC-ON2 preservation/consent | Existing valid state unchanged; drafts/preview zero writes/processes/network; optional templates no fixed model/effort; exact confirmation/apply/restart read-back; separately consented probes |
| AC-ON3 default-sdd | Historic r1/digests unchanged; accepted future r2 no effort; Codex-only/Claude-only unique resolution; both explicit selection supports mixed graph; unresolved multiplicity and disallowed Project policy block |
| AC-ON4 unavailable facts | Auth negative/unknown, model unknown/unavailable, effort unsupported/unproven and missing required task capability have separate actionable blockers; no ranking/fallback |
| AC-SEC lifecycle | Tombstone generation, stale preview/ABA/concurrency, fault/recovery/permissions/redaction tested; no portable/Workflow/Execution/credential rewrite on local removal |

Planning evidence is documentary only. Future deterministic/synthetic PASS cannot
stand for real model access, paid inference, live Provider effects or MVP acceptance.
