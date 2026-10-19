# Specification 007 amendment — Runtime admission without certification (AXM-36)

## Status and authority

Proposed, revision 1, 2026-10-19. **Specify only; human acceptance pending.**
Implements no behavior and grants no dispatch authority. The product direction
in [AXM-36](https://linear.app/rgomids/issue/AXM-36/simplificar-runtime-admission-e-tratamento-de-falhas-operacionais)
and its `READY_FOR_specify` comment are confirmed; they do not accept
[ADR-0032](../../decisions/0032-runtime-admission-without-certification.md).
Accepted contracts remain in force until their prospective reconciliation is
explicitly accepted. Plan/Implementation must wait for that decision; merge,
green checks and this document cannot supply it. No canonical Execution state
was available or reconstructed from the card's gate.

## Problem, actors and boundary

Previously, selection and dispatch require exact-version/digest registry
membership and fresh behavioral Capability Evidence, including model/effort
proof. A new installation can be refused before its requested operation runs.
The reported v0.20.0 / Codex 0.163.0 `subscription_observed` followed by
`capability_unknown` is an admission failure, not demonstrated Codex
incompatibility. The repository registry at the inspected base has no entries.

The future behavior operates the user's chosen Codex or Claude through its
adapter, within Project/Workflow/Execution authority, then diagnoses actual
failure. It does not promise vendor feature support, successful inference,
billing identity or sandbox enforcement.

Actors: the operator chooses Runtime/Profile and authorizes effects; the
Orchestrator/Agent requests work and reads safe results; Lingo resolves policy,
binds authority and supervises attempts; the adapter encodes the selected
invocation; the Runtime executes it; validators and the supervisor determine
the contract outcome. A child report is input, never acceptance or authority.

Non-goals: production code or file removal, schema/persistence changes in this
stage, real probes/inference, credential access, redefining Runtime or Model
Profile, redesigning supervision/coordination, a version catalog, broader MVP
scope, Provider mutations, merge, release or acceptance. PR #320 is historical
implementation, not this Work Item's delivery.

## RA-1 — Resolve intent without capability certification

1. Resolve an installed, available executable for the explicitly selected
   Runtime. Resolve an enabled local Profile, its model and explicit effort or
   `runtime-default` intent, Project Runtime Policy and Workflow/Agent binding.
   Preserve deterministic selection, ambiguity refusal and immutable Workflow
   binding. Project/Profile permission lists remain authorization policy; they
   are not executable version/digest allowlists.
2. Observe authentication using the existing bounded auth preflight and consent
   boundary. Missing, expired or unobservable authentication is actionable and
   cannot be replaced with assumed credentials. `subscription_observed` means
   observed login only; inference and billing remain unconfirmed.
3. Neither resolution, workflow preview/start, stage plan/run, retry nor resume
   may require `CapabilityProven`, ProfileProof, Capability Evidence, an
   approved version/digest, probe success or a qualification receipt. No
   substitute help-output, canary, model ping, attestation or mandatory support
   matrix may become another certification gate. Integration tests are
   engineering Evidence, not per-installation admission receipts.
4. Version is optional diagnostic observation, never eligibility. An unknown
   version alone cannot refuse execution. CLI/model/effort/Profile upgrades
   require no recertification. Invalid configuration or a confirmed policy,
   authority or safety conflict still refuses for its actual reason.
5. Profile remains the sole effort-intent owner. Preserve canonical adapter
   encoding, bounded inspection of recognized argv/environment/settings and
   refusal of conflicting overrides, malformed/unreadable configuration or
   unsafe paths. `runtime-default` requests no explicit selector and asserts
   no concrete vendor effort. No exact-version/digest declaration or
   `precedenceEstablished` certification is required to make an installation
   eligible. Adapter-owned source/precedence rules must be testable; a
   concrete ambiguity in the effective invocation is a configuration conflict,
   not a request to qualify the Runtime. An unfamiliar version alone is not
   such an ambiguity. Actual unsupported model/effort becomes an operational
   failure under RA-4. Do not claim an effective vendor level without Evidence.

## RA-2 — Bind authority and integrity, not compatibility

Preview discloses Runtime, Profile, model/effort intent, invocation mode,
executable identity, operation scope, repository/workspace/inbox roles,
limits, confinement facts/unknowns and local/external/inference effects.
Inspection follows existing consent requirements; preview writes no canonical
state and starts no inference. It never requires a development checkout,
test fixture registry, Capability Evidence store or probe envelope.

Retain exact preview digest, configuration/Execution revision checks,
authorization epoch and fresh dispatch-time validation. Executable identity
can bind resolved path and bytes to prevent silent replacement. A change
between preview and apply refuses the old authority before spawn and requests
a new preview; it never requests certification. If already committed authority
or a frozen Execution no longer matches, report drift and the supported
replan/recovery route; never rewrite historical bindings to the new identity.

Local effects, external/Provider effects and vendor inference require their
respective authority. Login, local inspection consent or an AI recommendation
grants none of the others. There is no hidden Runtime, Profile, model, effort,
API, authentication or permission fallback. Preserve least privilege,
credential isolation/redaction, path/symlink/ownership checks, canonical-state
protection, child inbox boundaries, process supervision, cancellation,
timeouts, budgets, locks and atomic publication/recovery. No capability proof
does not imply sandbox proof; report confinement honestly as observed,
including `unknown`, and preserve independently required safety controls.

## RA-3 — Operator and AI journey

`inspect → authorize → execute → observe → diagnose/recover`:

1. Inspect configuration, executable availability, auth observation and policy.
   Fix concrete blockers using bounded guidance; no prepare/probe workflow.
2. Preview the exact Stage operation and review identity, intent, limits and
   effects. Obtain the required human gates and explicit operation authority.
3. Apply that preview. Revalidate it, commit dispatch authority before spawn,
   then supervise the chosen Runtime. CLI acknowledgement of dispatch is not
   completion of the Stage.
4. Inspect persisted status, contract result and sanitized Evidence. Successful
   login, a child assertion or exit 0 alone never establishes task success.
5. Diagnose the specific failed boundary; reconcile uncertain effects before
   proposing retry. A configuration correction requires fresh preview and
   authority where applicable, not a qualifying probe.

CLI structured results and human/skill guidance must convey the same facts:
operation/attempt correlation, lifecycle outcome, failure class, confirmed
cause or explicitly labelled hypothesis, known/partial/unknown effects, safe
Evidence references, next action and required authority. A thin skill explains
Lingo's result and does not invent compatibility, mutate state from chat or
become a second source of truth. Exact wire names/exit mapping are Plan work;
the following semantic classes are normative, not new implemented commands.

## RA-4 — Operational failure and safe recovery

Preserve existing attempt outcomes (`blocked`, `failed`, `unknown`,
`cancelled`, `succeeded`) and terminal-save classification ownership. A cause
class does not override effect certainty: an auth error or timeout after spawn
can still leave unknown effects. Record the process exit code/signal separately
from the validated outcome; never normalize a failed contract into success.

| Failure class / observed boundary | Evidence and certainty | Operator + AI next action |
| --- | --- | --- |
| Executable unavailable / spawn failure | Safe lookup/spawn category; confirm no child started only when observed | Correct installation or executable configuration; preview again |
| Invocation option/interface incompatible | Bounded sanitized adapter diagnostic, exit/signal; do not infer solely from version | Explain rejected option/mode; correct the adapter/configuration through reviewed change; no silent downgrade |
| Authentication unavailable | Preflight category or actual auth rejection; login is not inference | Operator restores auth through vendor flow; re-observe and authorize; never expose tokens or switch to API |
| Filesystem/permission failure | Safe source/target role and errno category; distinguish preflight from post-spawn | Inspect ownership/path/symlink or required permission; correction needs its own authority, never auto-chmod |
| Model/effort unsupported | Selected intent and confirmed rejection if present; otherwise hypothesis | Operator chooses a permitted supported intent or reviews adapter correction; new preview, no auto-substitution |
| Timeout / cancellation | Deadline, termination observation and process outcome; unconfirmed stop remains unknown | Confirm process stop, inspect effects and reconcile before considering retry |
| Subprocess failure | Exit/signal and bounded sanitized diagnostic reference; cause may be unknown | Inspect diagnostic, determine effects and fix confirmed cause; do not guess from exit alone |
| Result absent/invalid | Missing qualifying record, validation failure or incomplete contract despite exit 0 | Inspect attempt inbox/validator Evidence and reconcile; never synthesize success from terminal prose |
| Partial/unknown effects or lost supervision | Observed effects, missing observations and reconciliation references | Stop automatic redispatch; reconcile with required authority before a separately authorized retry |

Evidence contains bounded categories, safe relative roles, correlation,
digests, observation provenance and confirmed facts; hypotheses are labelled
and never stored as confirmed causes. Do not retain raw chat, auth/config/env
contents, credentials, account identifiers, host paths or unbounded vendor
stderr. When safe diagnostics cannot establish a cause, state `unknown` and
name the smallest safe inspection needed. Preserve existing canonical result,
terminal rendering and host-presentation separation.

Retry uses existing budgets, fresh policy/auth/identity checks and explicit
attempt authority. `stage run` must not rerun completed or uncertain attempts.
An `unknown` attempt requires an appended reconciliation fact; live/orphan
process uncertainty blocks redispatch. Partial known effects also require
reconciliation of remaining work and authority, not blanket repetition. A
retry creates a new attempt and preserves history; reconciliation never
rewrites a failed/unknown attempt as success or silently widens effects.

## RA-5 — Verification contract (future implementation)

Run each applicable row for **both Codex and Claude**, through their production
adapter/composition. Use deterministic synthetic processes/auth fixtures; no
real inference or external effects are authorized here. Installed journeys
must use the installed binary and packaged skills outside the source tree.

| AC | Setup/action | Required observation |
| --- | --- | --- |
| RA-AC1 | Normal installed CLI, unregistered version, no registry/proof store/probe receipt; valid Profile, auth, policy, binding and authority | Workflow preview/start and stage plan/run reach the authorized synthetic child; valid result can succeed without certification |
| RA-AC2 | Upgrade CLI/model/effort before a new preview; then separately replace executable or change Profile between preview/apply | First case requires no recertification; second refuses stale authority with zero spawn and fresh-preview guidance |
| RA-AC3 | Disable Runtime/Profile, deny Project policy, mismatch Workflow/Agent binding or leave multiple eligible Profiles | Concrete deterministic blocker, no fallback and zero spawn |
| RA-AC4 | Omit local, external or inference authority separately; present observed login | Each unauthorized effect is refused; login grants no execution/inference authority |
| RA-AC5 | Missing/expired auth, absent executable, unsafe/symlink path, unreadable config or conflicting effort selector | Correct boundary/cause and actionable guidance, no repair or permission expansion; absent certificate/version entry alone never reproduces these blockers |
| RA-AC6 | Child rejects option, model or effort; child fails with exit/signal | Appropriate RA-4 class, preserved process result, confirmed cause only with diagnostic support; no alternate Runtime/model/API |
| RA-AC7 | Child exits 0 without qualifying result, or returns invalid result | Non-success contract classification and result guidance; no success from login/exit/prose |
| RA-AC8 | Timeout, failed termination observation, interrupted supervisor, partial writes or lost result | Honest effect certainty, persisted Evidence, no automatic redispatch; reconcile/live-process restrictions and budgeted authorized retry hold |
| RA-AC9 | Recognized effort override/configuration or child-view drift; attempt writes outside allowed scope/canonical state | Profile intent, inspection consent, filesystem/inbox/process/security controls still apply; no claim that removed proof provides confinement |
| RA-AC10 | Diagnostic contains credential/config/path/transcript payload; inspect CLI and skill response | Payload omitted/redacted; bounded correlated Evidence, labelled hypotheses and same actionable next step for operator/AI |

These are acceptance obligations, **not tests executed or runtime acceptance**.
Real Codex/Claude inference and external acceptance remain AXM-12/#278 with
separate authority. Integration success cannot certify sandbox enforcement.

## RA-6 — Verified coupling and prospective reconciliation

Inspected base: `de19d3279b1c9cbd730ca8bb2ab2f098e1b76d12` (origin/main).
Read-only remote check found no AXM-36 PR/branch before edits. Only directly
consuming contracts are mapped below; no repository-wide audit is claimed.

| Existing owner / confirmed dependency | Proposed bounded change after human acceptance |
| --- | --- |
| [ADR-0025 decision 5](../../decisions/0025-public-stage-dispatch-and-capability-proof.md#decision) | Retire behavioral proof, expiry/requalification and probe prerequisites for admission. Preserve authority, honest confinement reporting and sanitized Evidence; decisions 1–4 and supervision remain |
| [Binding amendment PM-5/PM-5a](amendment-276-axm22-binding-identity.md#pm--pre-mvp-canonical-configuration-format-bld-001b3c), PM-7 capability-proof references, PM.4 composition | Retire registry observer/lookup and proof selection as gates; retain epoch PM-6/PM-12, canonical configuration safety and independent current identity |
| Same amendment BI-2 items 2–6, BI-2a D2/D7 and items 2/5/9; PM-2 and PM-13 items 3–4/7 | Remove exact-version/digest declaration applicability and version-evidence prerequisites (including read-only variant qualification). Preserve Profile ownership, source inspection, no-override, consent, trusted roots and child equivalence. Do not invent keys or weaken path checks |
| [Effort amendment PE-03/04/05/05a/06](amendment-306-profile-effort-onboarding.md#2-accepted-contract-one-effort-owner) | Remove proof-dependent resolution, proof reference and model/token certification; separate current intent/source identity from qualification. Keep PE-01/02 ownership and deterministic selection. PE-05a exact-version evidence requirement changes; override/unsafe-source guards remain |
| [Dispatch amendment A2/A6](amendment-276-stage-dispatch.md) | Remove required-capability/model proof and proof reference as admission conditions; preserve required effect intent, exact authority/drift checks, A3 classification and coordination boundaries |
| `internal/runtimeadapter/{version_registry,binding_source,effort_applicability}.go` | Registry ErrBlocked and Evidence Selector currently feed current identity; separate observation from certification and remove obsolete producers structurally |
| `internal/runtimeadapter/{adapters,frozen_stage,stage_effort,effort_inspector}.go`, `internal/runtimeprofile/{v2,v3_consumer}.go` | Adapter effort proof checks, ProfileProof/ResolvedProfile mandatory proof/version fields and declaration matching require coordinated removal; retain adapter encoding/intent and safe inspection |
| `internal/capabilityproof/{service,observer,selection,evidence}.go`; `cmd/lingo/{capability_composition,runtime_policy,stage_dispatch,stage_service}.go` | Service/Observer enrich resolution with selected proof; stage composition and error mapping consume it. Retire certification services/commands/stores/tests only after ownership inventory; do not globally swallow ErrBlocked |
| `cmd/lingo/{execution_target,workflow_stage_plan}.go`; `internal/runtimeapplication/policy.go`; `internal/codexruntime/skills/{axiom-runtime,axiom-work-item}/SKILL.md` | Workflow preview/start targets and stage planning use the shared policy/observations; packaged guidance currently routes missing proof to capability commands. Reconcile these direct consumers without transferring authority to a skill |
| [#276 Plan/Tasks](plan-tasks-276-stage-agent-execution.md), [#306 amendment](amendment-306-profile-effort-onboarding.md), [CLI](../../commands.md) runtime capability and stage sections; packaged Runtime skills | Append prospective reconciliation of proof-related tasks/fixtures/guidance; retain historical Evidence. Replace mandatory show/prepare/probe/probe-extended journey with RA-3. Plan must trace workflow preview/start, stage plan/run/retry and supervisor per-attempt consumers end to end |

This is a change-impact boundary, not a schema design or approved Plan. No
blanket supersession of ADR-0024 or binding/digest formats is proposed:
ownership remains; fields that currently require proof need an explicit
versioning/retirement design rather than fake values or legacy reinterpretation.

## Acceptance gate, remaining decisions and constitution check

Before Plan/Implementation, a human must accept the exact ADR-0032 and this
amendment revision, including the bounded conflict map. Only then add the
ADR-0018 partial-supersession annotations to accepted ADR text/index and
prospective notes to affected Specifications. Do not rewrite historical text.
The `Supersedes` heading is deliberately deferred while the ADR is Proposed,
because the repository validator reserves it for effective supersession.

Plan must settle: proof-field/store/command retirement and supported historical
inspection without manufactured proof; tuple/format/version consequences;
adapter-owned effort-source/precedence representation without a per-version
qualification gate; optional version observation under existing subprocess
consent/limits; exact CLI taxonomy mapping and test ownership. These are
technical design obligations, not reasons to reopen the product decision.
No schema or persistence choice is authorized by this specify artifact.

Constitution check: Project != Repository (policy scope versus work target),
Role != Model (Agent role versus Profile selection), Execution != Agent
(durable attempt/gates versus actor), Runtime != Model (executable versus
selected inference identity), Skill != source of truth (thin guidance over
Lingo), Evidence != raw chat (sanitized facts/references). No invariant waiver,
new Provider effect, canonical gate transition or human acceptance is implied.

## Specify validation Evidence

At revision 1, repository bootstrap validation (`./scripts/validate-repository.sh .`)
passed, including ADR governance (33 ADRs), its 56 governance fixtures,
automation registration and sensitive-file checks. The standalone ADR checker
and `git diff --check` passed. An ephemeral deterministic check resolved all
13 relative links and heading anchors in the two new artifacts. Manual review
checked RA-1–RA-6 against the requested scope, failure classes, both-adapter AC,
partial-supersession boundaries and the six invariants above. No delegation:
the Runtime exposed model overrides but no verifiable strict capability
hierarchy relative to the Orchestrator, so the bounded inspection stayed local.

Go/runtime tests and RA-AC1–10 were not executed: this change defines future
behavior and changes no production behavior. No probes, inference or real auth
inspection ran. `gitleaks` is unavailable in this environment; repository
sensitive-file checks passed but do not replace a dedicated secret scanner.
Human acceptance and the coordinated structural retirement design remain open.
