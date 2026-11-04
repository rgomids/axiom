# Specification 015 amendment — Issue #369 pre-spawn launch

Revision: 1, 2026-11-04. Status: **Specified; AC-D material decision pending**.
Stage: Specify / attempt-01; no implementation or product acceptance claimed.
Canonical work: [Issue #369](https://github.com/rgomids/axiom/issues/369),
[approved four-area intake](https://github.com/rgomids/axiom/issues/369#issuecomment-6621735082).
Inspected base and remote main: `375960907ce1ba4799830e2f0a90ae978598ea23`.
Branch: `fix/369-launch-refused`; one PR, relation-only metadata.

## Authority and contract boundary

This is a bounded amendment to [Spec 015](spec.md), under accepted
[ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md),
especially decisions 1–5. Its original Proposed wording is historical;
ADR-0037 records subsequent human acceptance. Preserve the partial supersession
of [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) and
[Spec 007 #306 PE-05a](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md).
No new ledger, predictive admission, authentication prerequisite or ADR is needed.
Runtime != Model; Execution != Agent; Project != Repository.

Authorized scope is A–D and synthetic technical verification. This stage edits
only documentation. No personal configuration reads/edits, external Project
mutation, real inference, credentials, merge, release, acceptance or Issue close.
[Contribution](../../../CONTRIBUTING.md),
[tracker routing](../../development/linear-github-issue-tracking.md),
[security](../../../.agents/policies/security.md) and
[persisted-state quality](../../../.agents/policies/quality.md) govern delivery.

## Evidence and uncertainties

The Issue body and complete comment timeline, including intake acceptance and
published executor prompt, were read on 2026-11-04. The reported macOS incident
is source evidence, not a rerun here. No original host configuration was read.
At the inspected base:

- **A:** `internal/runtimeadapter/effort_sources.go`,
  `validEffortTOMLScalar` / `tomlEffortSelectorsBounded`, accepts only a narrow
  scalar grammar; arrays/inline tables fail before semantic selector inspection.
  `TestEffortSourcesTOMLNarrowGrammar` even classifies valid arrays, inline tables
  and dates as unsupported. `observeEffortFacts` maps parse failure to
  `ErrEffortConfigurationInvalid`, which also matches Unsafe through errors.Join.
  Thus the reported generic unsafe label is not proof of an unsafe file.
  `notify=[...]` alone must pass syntax/semantic inspection; with a real effort
  key it must still conflict. Exact vendor acceptance is untested and unclaimed.
- **B:** `internal/stageexecution/runtime_launcher.go:Build` returns
  `ErrRuntimeLaunch` for path, inbox, builder, root and selector errors;
  `engine.go:runAttempt` discards that error and calls `MarkSpawnFailed` with a
  constant. Build creates attempt/send/receive directories before selector
  inspection; a pre-spawn refusal is not necessarily free of local effects.
- **C:** `internal/projectapp/readiness.go` deliberately reports Runtime
  `selected_at_dispatch`. `internal/runtimeapplication/authoring.go` derives
  `ProfileView.Eligible` from configuration/allowlisting, without an actual
  workspace, child environment or frozen attempt. Neither proves admissibility.
  `BuildStageInvocation`, `StageEnvironment`, `StageEffortRoots.ResolveOperational`
  and `CheckFrozenSelectors` define the real local observation boundary.
- **D:** `PrepareAttempt` in `internal/workflow/execution_transitions.go` appends
  a numbered record before Build, and limits retry by `len(task.Attempts)`.
  `Execution.Validate` independently imposes the same bound; `MarkSpawnFailed`
  makes Build refusal indistinguishable from OS spawn failure. Default SDD r2
  intake sets maximumAttempts=2. `internal/local/execution_store.go` preserves
  terminal attempts and frozen identity through append-only transition checks.
  No inference or authentication happened in the reported pre-spawn failures.

Claude evaluation: `fixedEffortRules` declares `modelSettings`, `effortLevel`,
`maxEffortLevel` and the effort environment key. JSON inspection recursively
rejects their presence, including unknown per-model shapes. Existing
`TestOperationalEffortConflictsAtEveryFixedSource` asserts that behavior.
A Sonnet-specific value is **not established to override Haiku**, but neither is
its noninterference established for the invocation. The accepted ambiguous-source
rule therefore requires conservative refusal. Keep that result, expose
`selector_conflict`, and explicitly describe the uncertainty. Ignoring mismatched
model keys would require authoritative vendor semantics and a separate material
security decision; it is not part of this amendment's approved implementation.
Shell `subscription_observed` proves neither child auth nor successful inference.

## Acceptance matrix

All rows preserve frozen selectors, explicit authority, noninteractive child
environment, bounded output, path isolation and immutable prior Evidence.

| AC | Current → expected | Invariant | Observable validation | Negative security tests | Compatibility |
| --- | --- | --- | --- | --- | --- |
| A | Valid unrelated composites refused → valid bounded TOML accepted, real selectors refused | Syntax validity is separate from selector presence; no precedence inference | notify array alone passes; same document plus effort key conflicts; inline/nested/quoted/multiline forms agree | Duplicate keys/tables, malformed UTF-8/syntax, deep nesting, oversized files, symlink/identity races still refuse | Replace unsupported-valid test expectations; retain shared legacy scope rules and invalid/error checks |
| B | Opaque launch_refused → stable reason, phase, safe next action in CLI and persisted attempt | Never format raw error/config/path into diagnostics | Each Build exit maps distinctly; round-trip and reconciliation retain codes; unknown error has safe fallback | Canary path/value/token absent from text, JSON, logs, persisted bytes; joined invalid/unsafe classified most specifically | Retain broad launch_refused family; additive typed projection requires decoder/downgrade tests |
| C | Structural ready/eligible appears sufficient → scope disclosed plus optional invocation inspection | Inspection grants no authority and predicts no vendor/auth success | Same synthetic effective tuple gives same refusal in inspection and dispatch; all persistent bytes/directories unchanged | No subprocess, network, inbox creation, lease/CAS/update or Provider call; cancelled/unsafe/missing roots return blocked/unknown | Keep structural readiness/eligibility meaning; add explicit not_evaluated invocation/auth/inference dimensions |
| D | Build refusal consumes maximumAttempts → durable pre-execution refusal and separate bounded explicit recovery | No auto retry, evidence deletion, unknown replay or selector rebinding | 1 refusal then explicit corrected attempt succeeds; 2 refusals block further preparation; spawn failure/runtime failure remain distinct | Exhaust refusal/attempt bounds, stale fence/CAS, crash before terminal commit, unknown process, revoked authority, changed repository | HUMAN_DECISION_REQUIRED for prospective budget policy; no historical reclassification or migration |

## Minimum technical design

### A — bounded standards parsing (TECHNICAL_LOCAL)

Use a maintained TOML decoder for syntax and duplicate/table validation, then a
small bounded traversal of decoded keys/arrays/tables for semantic selector
presence. There is no TOML library in go.mod and Go has no standard TOML decoder.
The current hand parser cannot be repaired proportionally by adding only notify:
inline tables, multiline values, quoting and duplicate/table rules would remain
an expanding custom grammar. Do not use YAML/JSON conversion or run Python in the
product. Adopt one pinned Go TOML dependency during Plan after checking its source,
license, supported TOML version, advisories and enforced recursion/resource limit.
Candidate: stable `github.com/pelletier/go-toml/v2` decoder, not its unstable AST API.
No dependency is installed or accepted as architecture by this Specify artifact.

[TOML 1.0 specification](https://github.com/toml-lang/toml.io/blob/main/specs/en/v1.0.0.md)
allows arrays, inline tables and multiline arrays and prohibits duplicate keys.
The parser's [upstream recursion advisory](https://github.com/pelletier/go-toml/security/advisories/GHSA-frc4-6h9q-39fq)
and [release history](https://github.com/pelletier/go-toml/releases) make parser-time
bounds a required dependency-selection check, not something a post-decode walk
can fix. Plan must pin and verify the exact fixed version; if the candidate cannot
bound hostile decode before recursion/allocation, select another compliant bounded
decoder rather than quietly weakening protection.

Keep the existing 1 MiB/file, 64 managed drop-in limit, rooted no-symlink readers,
ancestor/file identity and double-read checks. Semantic traversal maximum depth
32 and 65,536 decoded nodes; parser-time nesting must be bounded at 128 or less
before recursive decode. Verify these limits with hostile fixtures before adoption.
Validity means valid TOML within these declared resource bounds, not arbitrary
size/depth. No values are logged. Recognize model_reasoning_effort at every
currently inspected path, including dotted/quoted/inline/array-table nesting;
mentions inside strings/comments are data. Do not merge equal-valued selectors.
Retain nestedProfiles=false restrictions for existing callers. Read/parse bounds
and unsafe source identity remain distinct from malformed configuration.

### B — carry safe typed reasons (TECHNICAL_LOCAL)

Keep launch_refused as the broad compatibility category; attach closed reason and
phase fields, using errors.Is/As at the source. Minimum reasons:
configuration_invalid, selector_conflict, configuration_unsafe,
path_unsafe, permission_denied, identity_unsafe, executable_unavailable,
builder_failed. Minimum phases: invocation_build, source_resolution,
selector_inspection, spawn, runtime. Cancellation stays cancellation, not refusal.
Classify joined Invalid before Unsafe. Use static redacted next-action messages
and only closed source classes (user/project/project-local/managed/drop-in).
No raw errors, values, host paths, arbitrary key names or child output. Unknown
errors map to builder_failed; do not guess permissions from ambiguous I/O errors.
Only claim permission_denied when typed cause establishes it.

Persist the reason/phase on the nested attempt; CLI status and reconciliation
project those facts rather than regenerating meaning from diagnostic strings.
If additive JSON cannot safely preserve format-4 interpretation, Plan must use
an explicit version boundary and stop for material compatibility authority.
No alternate diagnostic store or generic logging framework.

### C — read-only effective inspection (TECHNICAL_LOCAL)

Provide an optional `workflow stage diagnose` operation selecting Project,
Execution, stage and agent. Use the existing persisted frozen tuple for attempted
work; never silently re-resolve changed defaults. An explicit new-selection mode
may show a prospective tuple with fresh provenance and an unequivocal preview
label, applying the same resolver as explicit retry. It does not change the prior
snapshot. Missing context returns not_evaluated with a safe next action.

Split the pure invocation preparation/selector inspection from Build's inbox
materialization and reuse it at both boundaries. Inspection uses the same planned
paths and exact StageEnvironment/CWD/root rules without creating directories.
An absent would-be inbox cannot be reported as a created/permission-tested inbox:
report that check not_evaluated; validate existing ancestors read-only. Normal
Build must recheck actual materialized paths and selectors immediately before
spawn. Inspection is a point-in-time observation, never a reusable admission proof.

Output separate structural readiness, profile configuration eligibility,
invocation admissibility (admissible/refused/not_evaluated), authentication
(not_evaluated), and inference (not_run). Admissible means only the local checks
performed passed. Project validation/profile listings disclose that invocation
inspection was not performed and point to this optional operation. Do not inspect
all profiles, invent a mandatory preflight, call vendor status/version/auth, run a
builder that writes, or persist an inspection receipt. Explicit operation consent
covers only bounded recognized configuration reads, never credential/login files.

### D — prospective bounded recovery (HUMAN_DECISION_REQUIRED: HD-369-D)

Recommended decision, **proposed, not authorized by this document**:
maximumAttempts counts charged launch attempts that cross the spawn-call boundary,
including confirmed spawn failure, running failure and unresolved spawn outcomes.
It is an attempt budget, never proof that inference occurred. Separately allow
at most **2 deterministic pre-spawn refusal records per task**, lifetime total,
not reset by selection, runner restart, stage resume or successful launch.
Total nested records are bounded by maximumAttempts+2 (at most 12 with today's
10-attempt maximum). A third deterministic denial cannot create unbounded records:
refuse further dispatch before preparation when the refusal allowance is exhausted.
At that point require a fresh explicitly authorized Execution, without inherited
authority; never erase the old records. No automatic loop, even below either limit.

An explicitly resumed refusal appends a new attempt ID/number, prior-attempt link,
actor/time, frozen invocation/provenance, generation, authority and workspace
preserve disposition. Record Build outcome reason/phase and no-spawn evidence,
including local inbox preparation fact when applicable, in that same aggregate.
A successful Build must durably reserve a charged launch before the runner call;
a crash after preparation/reservation without terminal evidence is unknown and
requires existing reconciliation, never a free retry. No-spawn must be asserted
only by the fenced engine before calling Runner, never inferred from missing PID.
Admission revocation, cancellation and arbitrary unknown builder errors are not
eligible for free deterministic refusal credit; account conservatively under the
charged budget. Spawn failure has phase spawn; vendor rejection after start has
phase runtime. A diagnostic alone grants no authority to resume.

Use a frozen prospective budget-policy discriminator on new Executions; absent
means the existing total-record policy. Preserve existing format-4 terminal bytes,
counts, defaults, old exhaustion and unknown effects. Do not parse old diagnostic
strings to refund attempts, including the original incident. No migration,
conversion, old-format executor or in-place rebinding. Old binaries must reject
new policy bytes safely before effects; an upgrade/downgrade boundary test is
required. Plan must reconcile exact encoding with ADR-0037 and persisted-state
quality policy before implementation; an incompatible cutover needs explicit
human decision, not implied approval of HD-369-D.

Alternatives: retain total-record budgeting (simpler, but does not satisfy AC-D);
use only successful spawns as charged attempts (misleading inference budget and
unbounded spawn-failure risk unless another budget is added); retroactively refund
historical failures (requires unsafe inference from incomplete facts and conflicts
with immutable history). Recommend the prospective two-budget rule. Persisted
budget meaning and recovery behavior become costly to change after adoption.

## Reach, fixtures and next gates

Plan must enumerate callers before editing shared functions. Minimum reach:
`internal/runtimeadapter` effort source parser/rules/frozen invocation and tests;
`internal/stageexecution` launcher/engine, cancellation and diagnostic tests;
`internal/workflow` attempt schema/validation/transitions/ready/retry projections;
`internal/local` strict codec, protected store, immutable update and recovery tests;
`cmd/lingo` composition/status/retry and `internal/cli` route/help/JSON contracts;
`internal/projectapp` readiness and `internal/runtimeapplication` profile projections;
installed command documentation and affected canonical fixtures. Do not edit the
bound default-sdd-r2 definition to inflate budgets or rewrite its revision/digest.

Use temporary synthetic HOME/managed/repository roots and fake executable identity;
never invoke real vendors. Reuse existing effortInspector/operational fixtures.
Test TOML arrays/multiline arrays/inline tables/arrays-of-tables/date/string/comment
syntax; nested and equal-valued selectors; notify+selector independently; duplicate
and dotted-table conflicts; size/depth/node bounds; source replacement, symlink,
managed inventory drift. Test Claude synthetic Sonnet effort with frozen synthetic
Haiku as selector_conflict, including unknown modelSettings shape; no claim of
vendor model semantics. Check redaction canaries across CLI text/JSON/persistence.
Test read-only byte/directory snapshots and panic spies for runner/network/store
calls. Test distinct Build/spawn/runtime failures, both counters at boundaries,
exact retry lineage/new-selection, cancellation, CAS/stale runner, crash before and
after reservation, unknown reconciliation, historical policy and downgrade refusal.

Readiness: **READY_FOR_PLAN for A/B/C; AC-D implementation BLOCKED on HD-369-D**.
Plan/Tasks may proceed independently; no implementation in this Specify run.
No human ceremony is required for reversible local choices. The pending human
choice is the budget semantic change, not permission to publish these documents.
If approved later, append the decision reference; never rewrite acceptance history.
Implementation must then pass deterministic targeted checks, integrated review and
document reconciliation before any claim that the defect is corrected.

## Specify verification

Source/contract inspection supports root-cause analysis; it does not execute the
proposed behavior. Publication receipt on Issue #369 records exact check commands,
outcomes, commit, remote HEAD, PR, CI observations and next gate. Targeted existing
unit checks are baseline evidence only. Real vendor smoke/inference, original host
reproduction, product E2E and behavior verification of the proposed changes are
NOT_RUN. No implementation code, dependency, ADR or unrelated documentation added.
