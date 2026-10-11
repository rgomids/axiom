# Issue #306 — Local Runtime and Model Profile management Plan

## Status and authority

**Proposed; awaiting human review.** Only Planning is authorized. Tasks,
production implementation, vendor probes, merge, release, Issue closure and
product acceptance remain separately gated.

Technical owner: [Issue #306](https://github.com/rgomids/axiom/issues/306).
One branch, `feat/306-local-runtime-profiles`, and one Draft PR serve the entire
lifecycle. Human approval must identify the exact commit and artifact blobs;
material changes invalidate the affected approval.

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
[proposed contract](amendment-306-local-runtime-authoring.md), the
[security review](security-review-306-local-runtime-profiles.md), and
[proposed ADR-0023](../../decisions/0023-local-runtime-configuration-surface.md).
None is accepted by publication. No Tasks artifact is generated here.

## 1. Sources and current architecture

Source precedence is current executable behavior, accepted contracts, then
proposals. Issue #306 requirements and comments were read (no comments before
this execution); #305 has no implementation approval. #303's older body contains
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
| Local domain | `internal/runtimeprofile/runtimeprofile.go`: closed format 1, bounded codec/digest, named Profiles, `Resolver.Resolve` | Typed authoring requests, safe field projection and lifecycle invariants; keep sole resolver |
| Local persistence | `internal/local/runtime_profile_store.go`: `Create/Save/Load`, fixed `runtime-profiles/v1/configuration.json`, private roots/locks/publication | Preview-bound prior snapshot comparison inside locks, anchored pathname fences, safe removal and read-back |
| Project resolution | `internal/runtimeapplication/policy.go`: `projectConfiguration`, `Preview/Check/Availability` | Reuse, add precise diagnostic facts; never invent a second selection algorithm |
| Observations | `internal/runtimeadapter/observation.go`: exact executable digest and Axiom skill integration | Safe local reporting of installed vs available vs auth/capability unknown; no manufactured proof |
| Authentication | `internal/runtimeadapter/authpreflight.go`, `internal/graphapplication/runtime_policy.go` | Reuse separately consented status probes; preserve subscription rejection of credential references |
| Bootstrap | `internal/runtimebootstrap/bootstrap.go` discovers executables and installs Axiom integrations | Keep independent; it does not configure Profiles or credentials |
| CLI/application | `cmd/lingo/runtime_profile.go`, `runtime_policy.go`, `internal/cli/runtime_preview.go` | New authoring application methods composed in Lingo and declared in CLI; preserve existing resolution preview/validate behavior |
| Readiness | `cmd/lingo/readiness.go`, `internal/local/readiness_adapters.go`, `internal/projectapp/readiness.go` | Share local report with #231 and Workflow consumers; no global Doctor |
| Conversational guidance | Embedded `internal/codexruntime/skills/axiom-workflow/SKILL.md` supplies configuration readiness only | Proposed thin `axiom-runtime` local configuration owner; explicit contract/ADR acceptance required |
| Recovery/upgrade | `internal/local/publication.go`, `safe_fs.go`, `recovery.go`, `inventory.go`, stable corpus | Recognize Runtime/Profile recovery paths and removal protocol; preserve historical format/corpus |

Current Profiles already support arbitrary globally unique IDs and more than
three entries per Runtime. Bounds are eight Runtime entries and 32 total local
Profiles; actual adapters currently restrict Runtime IDs to `codex`/`claude`.
Portable Project policy independently retains eight Runtimes / 32 Profiles /
32 preferences and exact key/Runtime/model matching. No vendor ranking.

The v1 wire has Runtime-level `Enabled` and `CredentialReference`, Profile
model/capabilities/complexities, allowlists and local preferences. No Profile
`enabled`, arbitrary argv/environment, persisted effort default or per-Profile
credential is present. Profile eligibility can be changed by explicit allowlist
membership, without adding another persisted enabled flag.

## 2. Proposed scope and contract decisions

The [amendment](amendment-306-local-runtime-authoring.md) defines the proposed
authoring contract. Recommendations preserve local v1 and portable v1/v2/v3;
they are proposals until independently accepted.

| Decision | Recommended baseline | Alternative / cost |
| --- | --- | --- |
| D1 — first/last entries | Atomic initial Runtime + one or more Profiles; publish a typed tombstone at the canonical file when the reviewed compound change removes all entries | New empty-state format needs versioned codec/inventory migration; never silently loosen v1 validation |
| D2 — credentials | Author only vendor-managed subscription bindings with empty credential reference; recognize legacy nonempty references as opaque, redacted and blocked for subscription use | Add approved secure-reference backend later: source grammar, permission/effect review and non-subscription dispatch contract required |
| D3 — invocation/effort | Model + capabilities + complexities in existing wire; Workflow owns effort request; store valid effort declarations as unproven and block use without exact authoritative model/executable/version proof | Persist defaults or arbitrary invocation fields only through a separately accepted versioned extension; no vendor token translation |
| D4 — conversational owner | New thin configuration-only `axiom-runtime` skill, common Codex/Claude catalog, canonical CLI sole authority | CLI-only fails conversational acceptance; Project/Workflow ownership expansion contradicts accepted boundaries without a separate amendment |
| D5 — removal/replay | Explicit local cascade only; warn of dependent Projects without rewriting them; replay returns stale/conflict and requires fresh preview | Automatic repair/cascade/retry obscures authority; retained history store would introduce extra lifecycle obligations |

D1 permits retired *machine configuration* as a typed tombstone, not an invalid
empty v1 configuration. Active v1 configuration wire remains unchanged; the
new tombstone encoding requires independently accepted compatibility/recovery
handling (see amendment LA-03/LA-10). A Runtime entry cannot exist without its nonempty valid allowlist.
The initial create input is a bounded compound candidate so a clean machine can
add Codex and Claude, each with multiple Profiles, in one supported operation or
successive valid operations. Last-profile removal must explicitly include its
Runtime when it would otherwise leave an invalid allowlist; otherwise refuse
with actionable dependency diagnostics. Final configuration removal atomically replaces active configuration with a
secretless tombstone and does not delete private directories, install receipts,
credentials or Runtime binaries.

Globally unique Profile IDs remain the portable reference identity. Same display
idea across vendors uses distinct IDs, e.g. `codex-worker` and `claude-worker`.
No renaming or model changes propagate to Projects. No-op candidate edits return
unchanged without revision increment; replay of an old applied preview is stale.

G-3 authoritative model/effort proofs are independently owned by the existing
observation boundary in the accepted #303 amendment. Production observer proves
only `axiom-skills`; executable presence and configured declarations never prove
coding, effort or authentication. #306 removes authoring/configuration blockers,
not G-3, G-4 consented Provider linkage or final R-3 acceptance.

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
Runtime enable/disable changes only Runtime `Enabled`; Profile enable/disable
adds/removes its ID in that Runtime's allowlist. A disabled Profile can remain
defined; v1 cannot represent an empty allowlist, so final disable requires a
reviewed compound removal or reports that constraint. Local preferences remain
supported configuration but never override Project preferences.

## 4. Preview, apply, persistence and recovery

1. Read bounded protected state without creating files/locks/directories during
   preview. Classify absent/valid/unsafe/malformed/pending; malformed/pending is
   never treated as absence. Build safe candidate and explicit local effects.
2. Envelope binds operation, local target identity (opaque, not host path),
   configuration state (fresh/active/retired), format, numeric revision, canonical
   content digest, exact file/directory identity, candidate digest and full effects.
   For a missing root/suffix, bind the nearest existing protected ancestor and
   exact absent suffix; distinguish authorized creation from external appearance.
   Include local removals/allowlist/preference edits and known impacted Project
   references; bounded incomplete dependency discovery must be disclosed.
3. Apply requires `--expected-revision` (opaque snapshot token),
   `--preview-digest` and `--authorize-local`. Acquire existing ordered exclusive
   locks, re-read and compare the *reviewed* old bytes/digest/revision/identities,
   then recompute the same candidate/effects. Reject any drift or CREATE collision.
4. Assign revision 1 on fresh valid creation; active updates, retirement and
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
   rather than invoking fresh-absence Create or resetting revision. Never unlink
   generation metadata or introduce a second registry. Preserve unknown
   records and fail closed when the current protocol cannot prove a safe result.
7. Extend recovery discovery for `runtime-profiles/v1`, marker classification and
   valid canonical locations. A separately approved recovery envelope restores
   prior or finalizes committed state only when generation identity is certain.
8. Return safe read-back projection + committed revision/digest; independent new
   process reads must agree. A canonical pathname replacement after commit cannot
   be presented as confirmed success merely because an old opened root was safe.

The security review specifies threat controls and required fault injection.
There is no new registry, approval ledger or second filesystem authority layer.
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
rejection. Authentication observation alone does not prove model access or effort.

Reuse `runtimeapplication.Availability` for existing coarse Project readiness;
add scoped safe detail via the same local report, preserving existing result
contracts unless explicitly amended. Requirements-specific readiness still uses
the existing resolver and Workflow planning inputs. No automatic credentials
probe during `project validate`; no claim that #231 currently checks all model
requirements or authenticates vendors. Configuration changes invalidate future
resolution previews; immutable Execution/graph/Evidence bytes remain unchanged.

Two Projects may share identical local IDs/model binding. Removing or disabling
a local Profile leaves both portable policies intact, and their future matching
requests block. Project allowlist edits remain #305; Workflow references,
selection and stage execution remain their existing owners.

## 6. Compatibility, migration and release integration

Recommended baseline preserves active local v1 and portable schema versions.
It introduces a versioned tombstone record at the same canonical location, not
a second configuration store. Existing valid v1 files load unchanged; no normalization write on read, preview or validation.
Legacy credential references remain readable internally and safe-projected only;
authoring may explicitly clear them but never echo/copy their values into drafts.
Do not weaken decode/selection of historical configurations to satisfy new input
policy. Authoring safety applies before drafting and at apply boundaries.

Tombstone version 1 has closed `kind`, `formatVersion`, `revision` and
`priorConfigurationDigest` fields (amendment LA-03). New code reads active v1 or
tombstone; old code cannot interpret a tombstone and must fail closed. Downgrade
with retired state is unsupported until an independently approved restore creates
an active v1 generation; no automatic conversion. This FR-026 forward-path and
compatibility window must be explicitly accepted before implementation.

Changes to tombstone encoding/removal markers/recovery locations must update `StateRootDirectories`,
inventory classifiers and every-writer guards. Extend compatibility fixtures for
new supported protocol state; preserve stable-v1 snapshots byte-for-byte.
Follow quality-policy corpus freezing at release, not during Planning.
If implementation requires a new wire format, stop affected work and obtain a
versioned amendment/FR-026 compatibility window and upgrade path first.

New `axiom-runtime` embedded skill/catalog requires exact accepted inventory
extension, both Runtime installers, owned upgrade/retirement/conflict behavior,
history pinning and fresh/upgrade/reinstall validation. Historical receipts stay
immutable. Do not resurrect retired skill aliases or add maintainer runtime
configuration. Production docs/command reference and inventory parity change
only when approved implementation is delivered; no current-help promises now.

## 7. Sequencing, dependencies and parallel opportunities

This is implementation sequencing, **not generated Tasks**:

Contract/security/ADR decision → authoring boundary and safe store/recovery →
canonical CLI/composition → shared diagnostics → thin conversational integration
and installer parity → focused regressions/Evidence → independent review and
documentation reconciliation. Tasks require separate Plan approval.

Contracts and security investigation were delegated read-only because independent
surfaces benefited from bounded context. Both inherited orchestrator settings;
no higher model or hardcoded model selection. No runtime logical-profile-to-agent
model mapping is exposed, so no invented cheaper override is claimed. Findings
were checked against the referenced source. Subagents approved no artifacts.

After interfaces are approved, redaction tests/recovery fault design and skill
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

Planning validation is recorded in [the security review](security-review-306-local-runtime-profiles.md#planning-validation).
No production behavior is delivered by these checks. Future Evidence must name
exact head, commands, categories PASS/FAIL/BLOCKED/SKIPPED/UNAVAILABLE, synthetic
versus local versus native scope, and pending human acceptance.

## 9. Risks, human gates and stopping point

Main risks: last-entry removal protocol; weak legacy free-text credential/model
fields; pathname replacement; missing recovery discovery; incomplete dependent
Project inventory; observation proof gaps; skill inventory contract expansion;
overlap with #305/shared modules. Mitigations and refusal behavior are explicit
above and in the security review. No acceptance is inferred from green checks.

Human review must decide D1–D5 and independently accept the proposed Spec 002
authoring amendment/security review and ADR-0023 skill inventory extension.
Plan approval names its exact commit/blob and selected decisions. Generate Tasks
only after explicit Plan approval; implementation additionally requires Tasks,
required contract/ADR/security acceptance and explicit implementation authority.

Next: review the Plan and proposed decisions. **STOP before Tasks.** This PR
remains Draft through all intermediate phases. No merge, release, Issue closure,
Linear status/dependency changes or product acceptance is authorized here.
