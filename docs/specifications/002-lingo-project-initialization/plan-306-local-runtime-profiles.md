# Issue #306 — Local Runtime and Model Profile management Plan

## Status and authority

**Accepted, 2026-10-11**, by [explicit maintainer acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6109183466), fixed to commit `823bce3141bae5e90951c5239181c46d88cfbfe9` and blob `d4886e9c282ee12d3526566d0ddfc31f8fffb391`. This revision reconciles status only; the approved historical bytes remain in Git and the [acceptance receipt](evidence-306-acceptance-tasks.md). Tasks preparation is authorized; [Tasks](tasks-306-local-runtime-profiles.md) remain **Awaiting Human Approval — Tasks**. No implementation, vendor probes, merge, release, Issue closure or MVP acceptance.

Technical owner: [Issue #306](https://github.com/rgomids/axiom/issues/306). Preserve branch `feat/306-local-runtime-profiles` and Draft PR #310. Cross-cutting Workflow/compiler/Execution implementation ownership is still **Pending Human Decision**; the Tasks separate that bundle without expanding #306. Material contract changes require renewed exact acceptance.

## Investigation baseline

Main inspected on 2026-10-11: `5d52327b0367ba08d872b50e804df5c7d63a3e89`
(v0.16.0). Working tree was clean and detached. No existing #306 branch or
owning open PR was found before creating this branch from current main.

Reuse `internal/runtimeprofile`, `internal/runtimeapplication`, the protected
`internal/local/runtime_profile_store.go` and #231 Project readiness. Existing
local storage and selection do not constitute supported production authoring.
Investigation covers versioned contracts, exact preview/apply, credential
references, capability truth, persistence/recovery and conversational routing.

PR: [#310](https://github.com/rgomids/axiom/pull/310). The initial Planning
transition is recorded in [#306](https://github.com/rgomids/axiom/issues/306#issuecomment-6105235739).
The complete reviewed artifact set comprises this Plan, the
[accepted contract](amendment-306-local-runtime-authoring.md), the
[security review](security-review-306-local-runtime-profiles.md), and
[accepted ADR-0023](../../decisions/0023-local-runtime-configuration-surface.md).
Publication alone grants no acceptance. The later explicit acceptance above authorizes Tasks preparation; the initial Planning boundary remains historical.

## Maintainer reconciliation — 2026-10-11

Remote #310 HEAD was reverified as `2096509b03daa9fb345ec245cb9070bbb18a3479`
before any edits; main remained `5d52327b0367ba08d872b50e804df5c7d63a3e89`.
[Human D1–D5/onboarding direction](https://github.com/rgomids/axiom/issues/306#issuecomment-6108025323)
and [textual REQUEST CHANGES](https://github.com/rgomids/axiom/pull/310#issuecomment-6108025068)
replace the previous D3 recommendation. At that reconciliation the exact
revised artifacts remained Proposed; the later acceptance above accepts all six
contract artifacts and their version/migration window, not future code or behavior. The prior Plan
revision stays historical in Git, not authority for the new effort architecture.

Additional independently accepted artifacts: [Spec 007 effort/onboarding amendment](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md)
and [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md).
[Reconciliation Evidence](evidence-306-planning-reconciliation.md) records current
source facts, documentary checks and unchanged accepted/code artifacts.

## 1. Sources and current architecture

Source precedence is current executable behavior, accepted contracts, then
proposals. Issue #306 requirements and comments were read (no comments before
the initial Planning execution); #305 has no implementation approval. #303's older body contains
historical pending statements: its later acceptance records and current main
show accepted ADR-0022/amendment/R10 and delivered #308 in v0.16.0. Preserve
that history. #275 and #231 are closed prior deliveries, not new work owners.
The user's AXM-7 acceptance boundary is preserved; no Linear mutation occurs.

Read [Runtime Policy v2](runtime-policy-v2.md),
[Spec 002](spec.md), [#231 Evidence](evidence-issue-231.md),
[Spec 007](../007-configurable-workflows/spec.md), its
[accepted amendment](../007-configurable-workflows/amendment-303-workflow-skill.md),
[ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md),
ADR-0004/0008/0009/0014/0020 ownership references, quality/security policies and
[documentation governance](../../documentation.md).

| Area | Existing implementation | Missing / planned change |
| --- | --- | --- |
| Local domain | `internal/runtimeprofile/runtimeprofile.go`: closed format 1, bounded codec/digest, named Profiles, `Resolver.Resolve` | Typed authoring requests, safe field projection and lifecycle invariants; keep sole resolver; local v2 effort and versioned immutable binding |
| Local persistence | `internal/local/runtime_profile_store.go`: `Create/Save/Load`, fixed `runtime-profiles/v1/configuration.json`, private roots/locks/publication | Preview-bound prior snapshot comparison inside locks, anchored pathname fences, safe removal and read-back |
| Project resolution | `internal/runtimeapplication/policy.go`: `projectConfiguration`, `Preview/Check/Availability` | Reuse, add precise diagnostic facts; never invent a second selection algorithm |
| Observations | `internal/runtimeadapter/observation.go`: exact executable digest and Axiom skill integration | Safe local reporting of installed vs available vs auth/capability unknown; no manufactured proof |
| Authentication | `internal/runtimeadapter/authpreflight.go`, `internal/graphapplication/runtime_policy.go` | Reuse separately consented status probes; preserve subscription rejection of credential references |
| Bootstrap | `internal/runtimebootstrap/bootstrap.go` discovers executables and installs Axiom integrations | Keep independent; it does not configure Profiles or credentials |
| CLI/application | `cmd/lingo/runtime_profile.go`, `runtime_policy.go`, `internal/cli/runtime_preview.go` | New authoring application methods composed in Lingo and declared in CLI; preserve existing resolution preview/validate behavior |
| Readiness | `cmd/lingo/readiness.go`, `internal/local/readiness_adapters.go`, `internal/projectapp/readiness.go` | Share local report with #231 and Workflow consumers; no global Doctor |
| Conversational guidance | Embedded `internal/codexruntime/skills/axiom-workflow/SKILL.md` supplies configuration readiness only | Accepted thin `axiom-runtime` local configuration owner; implementation and inventory proof still required |
| Recovery/upgrade | `internal/local/publication.go`, `safe_fs.go`, `recovery.go`, `inventory.go`, stable corpus | Recognize Runtime/Profile recovery paths and removal protocol; preserve historical format/corpus |

Current Profiles already support arbitrary globally unique IDs and more than
three entries per Runtime. Bounds are eight Runtime entries and 32 total local
Profiles; actual adapters currently restrict Runtime IDs to `codex`/`claude`.
Portable Project policy independently retains eight Runtimes / 32 Profiles /
32 preferences and exact key/Runtime/model matching. No vendor ranking.

The v1 wire has Runtime-level `Enabled` and `CredentialReference`, Profile
model/capabilities/complexities, allowlists and local preferences. No Profile
`enabled`, arbitrary argv/environment, persisted effort or per-Profile
credential is present in current v1. Profile eligibility can be changed by explicit allowlist
membership, without adding another persisted enabled flag.

## 2. Accepted direction and future contracts

D1–D5 are confirmed conceptual directions, not pending choices among the former
options. Exact format, migration, design-security and Plan artifacts have independent acceptance.

| Direction | Accepted future design | Traceability |
| --- | --- | --- |
| D1 lifecycle | Valid active v2 creation; final same-file tombstone; generation increments through edit/migration/retirement/recreation; exact recovery and forward compatibility, no universal downgrade | Spec 002 LA-03/04/10; Spec 007 §3 |
| D2 auth | Existing Codex/Claude CLI subscriptions only; no token/session/login-file management or credential backend; observations/inference require separate authority | LA-06/08; Security credential section |
| D3 effort | Required effort in local ModelProfile v2; Workflow v2 forbids effort and indirect effort overrides; frozen resolvedProfile carries exact effort/proof | LA-07; Spec 007 PE-01–06; ADR-0024 |
| D4 skill | Thin axiom-runtime for local configuration, diagnostics and first-use guidance; application/CLI owns all effects | LA-11/12; ADR-0023 |
| D5 removal/replay | Known dependency effects + explicit inventory limits, exact snapshot/digest/identity checks; no Project/Workflow/Execution rewrite or fallback | LA-04/09; Security lifecycle cases |
| Onboarding | Optional fast/balanced/deep guidance, one valid Profile per installed Runtime enough structurally; preserve state, no fixed model/effort, no mandatory interactive installer | LA-12; Spec 007 ON-01–05 |

D1 represents retired machine configuration as a typed tombstone, not an empty
invalid file. The same canonical store decodes active v1/v2 and retired metadata;
no competing registry or version-directory selector. Keep the historical
`runtime-profiles/v1/configuration.json` location even for newer supported wire;
the directory name is a storage namespace, not implicit codec authority.
Creation accepts a bounded valid compound candidate with at least one Profile
per included Runtime. Remove last Profile only with an explicitly reviewed
valid cascade/Runtime removal; final removal publishes a tombstone, never deletes
credentials, binaries, receipts or directories. Profile eligibility uses existing
allowlists; final disable that would invalidate a nonempty allowlist blocks or
requires an explicit valid compound change.

Globally unique Profile IDs remain exact portable reference identities. Same
model/Runtime with different efforts uses distinct Profiles, never inferred tiers.
No automatic Project model/key/policy updates. No-op edits do not increment
revision; applied previews become stale and require fresh authority.

G-3 authoritative model/effort proof remains independently owned by the observation
boundary. Production observer proves executable identity and `axiom-skills`, not
model access/effort. G-2 config intent can be valid while operational readiness is
blocked. #306 removes authoring/setup blockers; G-3, G-4 and R-2/R-3 remain separate.

The Spec 007 amendment defines accepted future local v2, Workflow schema 2, Stage Plan/
compiler projection 2, configured Execution 3 and graph/proposal 2. This is an
explicit accepted migration contract, not an implied expansion into dispatch implementation.
Cross-cutting migration implementation ownership still requires an explicit human
assignment, in #306 or a separately authorized bounded Work Item; none is created.

## 3. Canonical operations and interfaces

Proposed CLI tree (names are proposed, not available commands):

| Operation | Proposed surface | Effect / scope |
| --- | --- | --- |
| Read/list/show | `runtime binding list/show`, `runtime profile list/show` | One machine-local store; safe typed projection; no Project required |
| Validate candidate | `runtime binding validate`, `runtime profile validate-candidate` | Bounded typed draft, structural/security checks and truthful readiness; zero vendor invocation |
| Create/edit/remove | `runtime binding create/edit/remove`, `runtime profile create/edit/remove` | Without authority: preview only; with exact revision/digest and local authority: apply |
| Recover | `runtime profile recover` | Existing recovery mechanism, exact scoped marker/objects and independent confirmation |
| Observe | Existing `runtime codex/claude status`, `runtime codex/claude auth` | Status/integration separate from consented auth subprocess; never inference |
| Resolve | Existing `runtime profile preview` and Project/work-item paths | Existing Project intersection; no authoring meaning added to this preview |

Preserve `runtime profile validate` no-argument contract and existing
`runtime profile preview` arguments/results. New validate-candidate distinguishes
draft validation from stored validation. Do not conflate the existing resolution
preview digest with the new mutation envelope. New results use canonical
completion/status/rendering and metadata so help/skill inspect reflect executable
arguments. Numeric storage revision is not sufficient as an authority token.

Extend `runtimeapplication` with local configuration use cases, keeping existing
Project policy service intact. Define a narrow local-store port for snapshot,
conditional create/update/remove and recovery; implement it only with the existing
store. CLI parses transport input; application owns candidate normalization,
revision assignment, effects/digests and authority; `runtimeprofile` owns domain
validation and selection. Do not expose `Save` as a direct user operation.

The application builds complete typed candidates from exact current state and
explicit partial edits; preserves unrelated entries; rejects duplicates/unknown
fields, malformed IDs, unsafe model values and arbitrary environment/commands.
Runtime enable/disable changes only Runtime `Enabled`; local v2 Profile authoring
requires explicitly chosen effort; Profile enable/disable
adds/removes its ID in that Runtime's allowlist. A disabled Profile can remain
defined; v1 cannot represent an empty allowlist, so final disable requires a
reviewed compound removal or reports that constraint. Local preferences remain
supported configuration but never override Project preferences.

## 4. Preview, apply, persistence and recovery

1. Read bounded protected state without creating files/locks/directories during
   preview. Classify absent/valid/unsafe/malformed/pending; malformed/pending is
   never treated as absence. Build safe candidate and explicit local effects.
2. Envelope binds operation, local target identity (opaque, not host path),
   configuration state (fresh/active-v1/active-v2/retired), format, numeric revision, canonical
   content digest, exact file/directory identity, candidate digest and full effects.
   For a missing root/suffix, bind the nearest existing protected ancestor and
   exact absent suffix; distinguish authorized creation from external appearance.
   Include local removals/allowlist/preference edits and known impacted Project
   references; bounded incomplete dependency discovery must be disclosed.
3. Apply requires `--expected-revision` (opaque snapshot token),
   `--preview-digest` and `--authorize-local`. Acquire existing ordered exclusive
   locks, re-read and compare the *reviewed* old bytes/digest/revision/identities,
   then recompute the same candidate/effects. Reject any drift or CREATE collision.
4. Assign revision 1 on fresh valid v2 creation; active updates, migration, retirement and
   recreation increment the persisted generation exactly once, rejecting overflow.
   Tombstones preserve generation across supported remove/recreate cycles. An
   existing empty version directory is incomplete state, never fresh absence;
   unknown or externally missing generation requires explicit reconciliation.
   Do not claim detection of arbitrary erased history outside protected state.
5. Retain `AnchoredDirectory` identity through staging and publication. Check
   `StillAtPath` before commit and after it; publication failure after possible
   effects yields uncertainty/recovery, never success or automatic retry.
6. Reuse bounded private reads, cross-platform permission/ACL checks, lock order,
   atomic publication, digest-bound markers, fsync and exact read-back. Final
   removal publishes the strict tombstone through the same single-file atomic
   publication protocol; recreation conditionally replaces that reviewed tombstone,
   with active v2, rather than invoking fresh-absence Create or resetting revision. Never unlink
   generation metadata or introduce a second registry. Preserve unknown
   records and fail closed when the current protocol cannot prove a safe result.
7. Extend recovery discovery for `runtime-profiles/v1`, marker classification and
   valid canonical locations. A separately approved recovery envelope restores
   prior or finalizes committed state only when generation identity is certain.
8. Return safe read-back projection + committed revision/digest; independent new
   process reads must agree. A canonical pathname replacement after commit cannot
   be presented as confirmed success merely because an old opened root was safe.

The security review specifies threat controls and required fault injection.
The accepted schema/migration contract requires separately versioned codec/inventory
support, not silent relaxation of an accepted mapping.
There is no new registry, approval ledger or second filesystem authority layer.
Committed rollback is a new monotonic reviewed generation, not restoration of an
old revision. Pre-commit crash recovery can restore the uncommitted prior bytes.
Removal rollback is separately previewed recreation/restoration, not automatic
reinstallation, credential recovery or modification of existing Executions.

## 5. Authentication, capabilities and readiness

Store declarations remain distinct from observations. Read/list/show expose
structural validity, enabled/allowlisted eligibility, installed/integration facts,
and auth/model/effort `unknown | unsupported | observed` distinctions with
provenance. Safe blocker classes include missing configuration, unavailable or
disabled Runtime/Profile, exact model mismatch, auth unproven/negative,
unsupported effort, ambiguous intersection, stale authority and recovery needed.
Reason codes and logical references must be actionable without leaking input.

Default validation/preview/apply never invokes a vendor, reads credential values,
logs in, refreshes login or mutates Providers. Existing auth preflight may run
only after an explicit consented observation action; observation authority does
not authorize inference. Bind auth facts to exact executable and effective
invocation environment; do not persist a previous auth result as execution
permission. Subscription dispatch retains fresh preflight and nonempty-reference
rejection. Local v2 authoring cannot populate a credential-reference field; legacy
v1 references are diagnostic-only and never resolved/copied by this feature. Authentication observation alone does not prove model access or effort.

Reuse `runtimeapplication.Availability` for existing coarse Project readiness;
add scoped safe detail via the same local report, preserving existing result
contracts unless explicitly amended. Requirements-specific readiness still uses
the existing resolver and Workflow planning inputs. No automatic credentials
probe during `project validate`; no claim that #231 currently checks all model
requirements or authenticates vendors. Configuration changes invalidate future
resolution previews; immutable Execution/graph/Evidence bytes remain unchanged.
Effort comes solely from selected local v2 Profile, never Workflow Agent or generic
WorkUnit controls. Its exact mode/value, Profile/config/proof digests and version/
mode tuple freeze at each authorized resolution/admission, then dispatch checks
that frozen tuple. Unknown model/effort capability blocks; no substitute choice.

Two Projects may share identical local IDs/model binding. Removing or disabling
a local Profile leaves both portable policies intact, and their future matching
requests block. Project allowlist edits remain #305; Workflow references,
selection and stage execution remain their existing owners.

## 6. Compatibility, migration and default onboarding

The canonical [Spec 007 amendment §3](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md#3-versioning-and-explicit-migration-strategy)
defines the full version matrix, FR-026 window and forward paths. New readers
preserve local v1, Workflow v1 and supported historical Execution/graph bytes for
safe inspection; no automatic codec normalization or effort default inference.
Local v1 effort is unconfigured for future v2 admission. Explicit local migration
chooses effort and increments generation. Workflow migration publishes a new
immutable schema-2 revision with user-approved Profile mapping; reports historical
Agent effort beside the candidate Profile effort. Local creation, portable policy
edit, definition publication, Project active selection and Work Item start have
separate authority. Legacy executions without v2 snapshots cannot gain new effort
through resume/retry; safely block unsupported continuation, preserve all records.

Local format 2 removes authored credential-reference support and requires effort.
Legacy nonempty references stay redacted/readable internally; migration never copies
or resolves them, and refuses unless the reviewed candidate explicitly clears them.
Do not scan vendor login files or convert their secrets into Axiom configuration.

MVP does not promise universal old-reader compatibility. Older readers fail safely
on newer formats/tombstones. Downgrade with such records is unsupported; no automatic
lossy conversion/deletion. Preserve stable corpus/receipts and extend canonical
inventory/every-writer tests plus new-version fixtures; no weakening old guards.
Recovery only restores/finalizes provable exact states with scoped confirmation.

Onboarding is specified in [ON-01–05](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md#4-onboarding-and-default-sdd-journey):
noninteractive installation; passive installed Runtime inventory; axiom-runtime
first-use guidance; optional fast/balanced/deep suggestions with no concrete model
or effort; one valid Profile per detected Runtime; precise unknown blockers;
explicit local confirmation and read-back. Reuse detection without invoking
runtimebootstrap.Run as a read operation, because Run installs Axiom integrations.
No absent Runtime binding and no credential/probe effects.

Default-sdd r1 and its digest remain historical/addressable. Accepted future r2/schema2
has no Agent effort, vendor/model preference or universal effort token. The user
confirms an explicit Profile assignment in a Project-owned copy or a separate
portable role/complexity preference. policy-default reuses the sole resolver;
Codex-only, Claude-only and both are supported through choices, not Runtime order.
Two eligible unresolved candidates still block. Existing Project policy edits
remain #305; readiness blocked by them is reported rather than auto-corrected.

New embedded axiom-runtime skill/catalog/inventory has ADR-0023 exact acceptance;
both Runtime fresh/upgrade/reinstall/retirement/parity checks and historical pin
preservation remain required. Future schema/default fixture changes occur only during authorized
implementation; this reconciliation edits neither schemas nor production fixtures.

## 7. Sequencing, dependencies and parallel opportunities

This is the accepted implementation sequencing; the [proposed Tasks](tasks-306-local-runtime-profiles.md) now decompose it under separate review:

Contract/security/ADR and migration ownership acceptance → versioned local/Profile
and Workflow interfaces → authoring boundary and safe store/recovery →
canonical CLI/composition → shared diagnostics → thin conversational integration
and installer parity → focused regressions/Evidence → independent review and
documentation reconciliation. Plan approval is recorded; exact Tasks approval remains required.

Contract, security and onboarding investigation/review were delegated read-only
where independent surfaces benefited from bounded context. All inherited
orchestrator settings;
no higher model or hardcoded model selection. No runtime logical-profile-to-agent
model mapping is exposed, so no invented cheaper override is claimed. Findings
were checked against the referenced source. Subagents approved no artifacts.

After interfaces are approved, documentary security review, onboarding scenarios
and compatibility design can proceed independently. Future implementation of
redaction tests/recovery fault design and skill
guidance can be developed independently of CLI parsing. Store/recovery changes
share filesystem primitives and must serialize; resolver/CLI metadata and
installed inventory edits each need one owner. Integrate only against the same
branch and PR. Inspect current main/open PR overlapping `runtimeprofile`,
`runtimeapplication`, `local`, readiness and skill catalog before implementation
and whenever upstream changes; no currently open PRs were found at baseline.
Coordinate #305 shared readiness work without waiting for its policy edits.

## 8. Verification and acceptance mapping

| #306 acceptance | Planned inspectable Evidence |
| --- | --- |
| Clean state; both Runtimes; multiple named Profiles; conversational configuration | Black-box fresh isolated state configure Codex + Claude with at least four Profiles each; CLI and synthetic skill prepare/preview/confirm/apply/read-back; no manual state/credential authoring |
| Exact Project intersection and precise fail-closed diagnostics | Existing resolver/project policy regressions plus missing, disabled, mismatch, ambiguous, negative/unproven auth and capability cases; no fallback |
| Two Projects share; remove without rewriting immutable state | Snapshot Project/Workflow/Execution bytes; approved local removal; fresh readiness/selection blocked for both; repair via new preview restores future candidates |
| Truthful model/effort/auth observations | Deterministic injected sources with stale executable/model/version proof, unknown/unsupported effort, negative auth; separately label real status probes and G-3 limits |
| Reviewed mutation authority; no secret leakage | Absent/conflicting authority, stale same-revision content, file replacement/ABA, CREATE collision, cross-process contention, symlink/hardlink/permissions/ACL, redaction/injection, preview zero-write/process/network spies |
| Independent contracts/security; no Doctor | Exact artifact acceptance record in existing PR/Issue required before implementation; tests confirm existing readiness reuse and no new aggregate Doctor route |
| G-2 blockers specifically removed | Repeat #303 Lane L/#275 C–G preparation using production authoring rather than fixture-created config; record each G-2 result and independent G-3/G-4 blockers; no automatic R-3 acceptance |

Future checks: focused `go test` for `runtimeprofile`, `runtimeapplication`,
`runtimeadapter`, `local`, `projectapp`, CLI and Lingo; then affected graph/skill
installation suites. Security concurrency tests use `-race` where supported;
`go test -race ./...`, `go vet ./...`, `go build ./...`, repository validators
and native Windows permission/recovery/CLI suites complete implementation CI.
Fake observation PASS is not native auth/model inference. Live Codex/Claude
conversational R-2 and full R-3 remain #278 with their own authority.

Additional revised acceptance is [AC-PE1–3 / AC-ON1–4 / AC-SEC](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md#5-scope-ownership-and-acceptance):
sole effort owner, explicit migration/history, immutable snapshot/drift proof,
none/Codex-only/Claude-only/both first use, one/three/more named Profiles, template
neutrality, restart/read-back, preserved existing config and unproven/unsupported
facts. Map all to classified future Evidence, never native proof from synthetic PASS.

Current reconciliation validation is recorded in [Evidence](evidence-306-planning-reconciliation.md).
The [security review](security-review-306-local-runtime-profiles.md#planning-validation)
separates prior-baseline checks from current documentary validation.
No production behavior is delivered by these checks. Future Evidence must name
exact head, commands, categories PASS/FAIL/BLOCKED/SKIPPED/UNAVAILABLE, synthetic
versus local versus native scope, and pending human acceptance.

## 9. Risks, human gates and stopping point

Risks: new active v2/retirement/Workflow/Execution codecs; legacy continuation;
unknown model/effort proof; shared Project readiness/#305 dependency; incomplete
Project inventory; first-use templates mistaken for executable configuration;
concurrency/identity/recovery and redaction. Controls are explicit in both
amendments and security review, with objective future acceptance criteria.

D1–D5/onboarding and all six exact artifacts are accepted through [explicit maintainer acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6109183466).
The version/migration/legacy-continuation window and runtime-default no-override
semantics are accepted design contracts, not undecided conceptual choices.
Implementation ownership of the cross-cutting Workflow/compiler/Execution bundle
remains Pending Human Decision. The [Tasks](tasks-306-local-runtime-profiles.md)
recommend separate bounded ownership, without creating an Issue.

Tasks require exact human approval; implementation additionally requires explicit
bounded implementation authority and independent future-code security review.
No CI/document/synthetic result proves live Runtime operation or accepts the MVP.
Next: review Tasks and ownership. **STOP before Implementation.** PR remains Draft;
no merge, release, Issue closure, Linear transition or product acceptance here.
