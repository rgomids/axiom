# Specification 012 — Tasks r1 (AXM-39)

## Implementation checkpoint — 2026-10-23

Same PR/branch, implement **attempt-04 PARTIAL**. [Implementation Evidence](evidence-implementation.md#attempt-04--manual-host-update-preflight)
records actual manual-update read-back: Axiom 0.21.1 / 921507c27c5b / clean,
but three installed skills in both roots precede approved 012 guidance;
Work Item matches. Docs inspection exists but refuses **source_unqualified**.
Both Axiom auth preflights refuse environment overrides, despite direct login
availability; inference unobserved. No installation retry or protected-state change.
T07 **BLOCKED_HOST_BASELINE**, both native conversations **NOT_RUN**, loaded UNKNOWN.
T01–T06/E01–E06 historical PASS preserved; T08/E08 PARTIAL; AC1–AC5 PARTIAL,
AC6 PASS_SCOPED_REVIEW. Next gate:
**resume_implement_after_verified_host_baseline_and_auth**;
**AWAITING_EXECUTION_REPORT**. Same-attempt callback records exact pushed SHA/CI.
Historical Tasks baseline below remains unchanged.

## Historical Tasks baseline, scope and execution state

2026-10-23; TASKS / attempt-01. Decomposition of [Spec 012 r1](spec.md),
[C01–C16](examples.md) and [Implementation Plan r1](plan.md), preserving six AC.
[Human Intake baseline](https://linear.app/rgomids/issue/AXM-39#comment-70d285fc)
and [documentary review](https://github.com/rgomids/axiom/pull/343#pullrequestreview-5628134742)
confirmed against canonical [Draft PR #343](https://github.com/rgomids/axiom/pull/343),
branch `docs/axm-39-conversational-presentation`, source HEAD
`c723430d662e4ea3761ffd5ae2eebeec716b5166`. Local/remote matched before writing.
GitHub owns technical artifacts; Linear owns operational events. Prior stage
stopping statements remain historical. This stage produces documents only.
All AXM39-T01–T08 execution statuses are **NOT_STARTED**; product checks **NOT_RUN**.
The Tasks artifact can complete while its future implementation Tasks remain open.

Preserve [007](../007-configurable-workflows/spec.md),
[008](../008-output-separation/spec.md), [011](../011-rendered-workflow-presentation/spec.md),
[AXM-36 RA-1–6](../007-configurable-workflows/amendment-axm36-runtime-admission.md),
[ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md),
[ADR-0032](../../decisions/0032-runtime-admission-without-certification.md) and
[security policy](../../../.agents/policies/security.md).
DESIGN=NOT_APPLICABLE: conversational guidance, no new visual product surface or
design system; host legibility, existing diagram behavior and text fallback still apply.
No domain/CLI JSON/text/flags, Workflow/Execution, Registry, digest, consent,
Runtime admission, new formatter/producer/framework, private plugin, card, PR or ADR.
AXM-12 retains real inference/E2E and human acceptance. DF-01 remains
**DEFERRED_NOT_IMPLEMENTED**, never a Task dependency or added requirement.

## Ownership, order and shared Evidence rules

Owner labels below identify technical responsibilities/packages, not staffing or
permission grants. Executor/Orchestrator owns scope, authority, fan-in, checks and
Evidence; direct sequential execution while lower subagent capability is unproven.
Paths are impact bounds: inspect first; change only demonstrated missing guidance.
No concurrent writers to shared skills or manifest/history, even across work items.

| Task | Causal prerequisites | Execution class / exit |
| --- | --- | --- |
| AXM39-T01 | Implementation entry and current source/authority check | Local read-only preflight, exact source and ownership inventory |
| AXM39-T02 | T01 | Local test-only fixtures, semantic oracle and regression baseline |
| AXM39-T03 | T02 | Local Project guidance; serialized shared-file owner |
| AXM39-T04 | T03 (serialization), T02 | Local Stage/Runtime/readiness guidance |
| AXM39-T05 | T03 + T04 final bytes; T01 replaced-revision inventory | Local packaging, full four-skill history and rollback checks |
| AXM39-T06 | T02 + T05 | Local synthetic/package/security review |
| AXM39-T07 | T06 + installed read-back + action-specific authority | External Codex and Claude conversations; otherwise NOT_RUN |
| AXM39-T08 | T06 for partial handoff; T07 for complete product Evidence | Evidence reconciliation, implementation/review handoff |

DAG: T01→T02→T03→T04→T05→T06→T07→T08, with the additional joins above.
T03→T04 serializes Project edits shared with readiness work; no parallel branch
or duplicate file owner. T08 may publish honest partial Evidence without T07;
it cannot declare AC5 or implementation complete. No human transition approval
is required. External action gates apply only to that action, not safe local work.
This invocation stops at Tasks; a verified orchestration handoff governs Implement.

Each future Task records input commit/blob/revision, owner, scoped diff, actual
command and exit/result, case/AC, sanitized artifact link and limitations in
`docs/specifications/012-conversational-project-stage-presentation/evidence-implementation.md`
(future artifact, not created here). Preserve failures and PENDING/NOT_RUN.
Instruction checks prove packaged guidance; semantic candidates prove only offline
oracle behavior; temporary roots prove packaging; installed read-back proves only
observed installation; native sessions prove only observed host behavior; human
acceptance remains separate. No retroactive Evidence or historical fixture edits.

## AXM39-T01 — Source, dependency and shared-writer preflight

**Owner:** executor/Orchestrator; `internal/codexruntime` integration owner.
**Preconditions/dependencies:** implementation entry, same PR/branch; no Task predecessor.
**Paths:** this directory's Spec/examples/Plan/Tasks; `internal/codexruntime/manifest.go`,
`integration.go`, `shared_history_test.go`; four `internal/codexruntime/skills/axiom-{project,workflow,runtime,work-item}/SKILL.md`.
Read `internal/cli/cli.go`, `internal/projectapp/{list,readiness,workflow_authoring}.go`
for canonical producers, without edits.
**Work/limits:** re-read PR metadata, exact remote/local HEAD, current AXM-39 events
and AXM-36/37/38 delivery plus active overlapping writers. Current Tasks preflight
found dependencies Done and AXM-12 In Progress; these are not installed-host facts.
Capture complete current four-skill bytes/digests, manifest identity, history and
compatibility 3 before any skill edit; preserve original set for T05. Reconcile
source drift or concurrent ownership before dependent writes; never overwrite.
**Done/negative:** exact baseline and dependency source features present, clean or
explicitly preserved user changes, no overlapping writer, concrete source inventory.
Drift, missing delivery or unresolved overlap holds affected writes; Done/CI alone
cannot certify installation or native behavior. Do not read user-global roots here.
**Evidence:** E01 source/authority/dependency read-back and four-skill inventory.

## AXM39-T02 — C01–C16 test-only semantic coverage

**Owner:** `internal/cli` test owner with `internal/codexruntime` oracle owner.
**Preconditions/dependencies:** T01 source inventory accepted locally.
**Paths:** `internal/cli/output_separation_test.go`,
`internal/cli/workflow_presentation_fixture_test.go`,
`internal/cli/testdata/presentation/` (new 012 specimens),
`internal/codexruntime/workflow_presentation_test.go`,
`internal/codexruntime/workflow_skill_content_test.go`; inspect existing
`internal/codexruntime/testdata/rendered-workflow/` fixtures without rewriting them.
**Work/limits:** reuse Go testing, typed DTO inputs and `semanticObservation` /
`semanticClaims` / `checkCandidate` conventions. Existing helper lacks complete
Project/Stage/Profile coverage; narrowly extend test-only claims or case-specific
assertions without weakening existing completion/effects/reference checks. Ordinary
name-only narratives must not inherit mandatory all-ID prose. Missing and explicitly
empty stay distinct. Keep invalid-source fixtures explicitly negative. Reuse
`TestWorkflowShowSyntheticPresentationEnvelopeMatchesProducer`; add targeted
producer equality where a new valid fixture needs it. No production mapper/schema.
**Done/negative:** every C01–C16 has a positive and omission/invention mutant;
checker rejects dropped blockers/fields, wrong full ID/digest, invented name,
selection/execution/model/effort, suppressed uncertainty, unsafe retry, fake rendering
and injected instructions. Change canonical input facts and require changed claims.
Use independent expected facts, not the same transformation for actual and oracle.
Record regression-first guidance tests failing on unchanged guidance where applicable;
positive oracle tests alone need not fail before guidance changes. Both Stage summaries
and requested advanced detail have assertions; natural and explicit entries covered.
**Evidence:** E02 case ledger with positive/mutant IDs, actual focused Go results,
producer-shape review and before/after guidance regression results; synthetic only.

## AXM39-T03 — Project labels and exact operational identity

**Owner:** `internal/codexruntime` packaged axiom-project guidance owner.
**Preconditions/dependencies:** T02 reviewed identity oracles; exclusive shared-file writer.
**Paths:** `internal/codexruntime/skills/axiom-project/SKILL.md`,
`internal/codexruntime/workflow_skill_content_test.go` and T02 tests/specimens.
`internal/cli/cli.go` and `internal/projectapp/list.go` remain read-only.
**Work/limits:** P1–P2 for list/show/validate and natural follow-up: name+slug,
one field, no fields, collision, duplicate same-ID records, partial inventory and
Unicode/control escaping. ProjectView has no name; correlate existing observed list
metadata by exact ID and compatible source or use honest slug/ID fallback. No new
wire field, generated name, UUID prefix algorithm, order change or global uniqueness.
Preserve full selectors, copyable syntax, references, consent and digest bytes;
unique supported slug remains usable. Labels are inert data, never shell fragments.
**Done/negative:** C01–C06/C16 positives and mutants pass; full requested ID and
collision IDs retained, no first-match choice or identity normalization, unresolved
identity stops dependent sensitive action. Test-only samples cannot prove AI compliance.
**Evidence:** E03 guidance diff and C01–C06/C16 results referencing E02; installed
and native results remain separate. T04 receives final Project bytes for serialization.

## AXM39-T04 — Progressive Stage/Agent and truthful Runtime/readiness

**Owner:** `internal/codexruntime` Workflow/Runtime guidance owner; Project owner
for readiness alignment; Work Item owner reviews cross-entry handoff.
**Preconditions/dependencies:** T02 semantic baseline and T03 final Project edits.
**Paths:** `internal/codexruntime/skills/axiom-workflow/SKILL.md`,
`internal/codexruntime/skills/axiom-runtime/SKILL.md`,
`internal/codexruntime/skills/axiom-project/SKILL.md`;
`internal/codexruntime/skills/axiom-work-item/SKILL.md` only if review/test shows a gap;
T02 content/oracle tests and specimens.
**Work/limits:** S1–S2/R1–R3: both actual Stages show purpose, I/O, gates/timing,
Role/Agent, Runtime/Profile certainty and completion; advanced validators, retries,
limits, timeout, concurrency, effect ceilings and failure policy recoverable from
same revision. Promote material risk/blockers immediately. Preserve AXM-38 ordered
Stages, same-Stage DAG, full reference, selection and accessible text fallback.
Distinguish allowed/declared/selected/resolved/observed, policy-default/codex-test,
Profile-owned effort and stale/missing facts. Preserve RA-1–6, including no admission
certification/probe/registry/allowlist or hidden Runtime/Profile/permission fallback.
**Done/negative:** C07–C16 pass positive/mutant checks. Success+blocked states exact
blocker/scope/next action; “not performed” requires no-dispatch facts. Multiple blockers,
prior attempts/unknown effects, invalid result despite exit 0, login-only and unknown
version stay honest. Reconcile before retry; no invented host preference or rendering.
**Evidence:** E04 per-skill applicability/diff, C07–C16 results and source attribution.
Record Work Item reviewed unchanged if no demonstrated missing guidance.

## AXM39-T05 — Complete shared history, both roots and rollback

**Owner:** `internal/codexruntime` packaging owner; `internal/runtimebootstrap` consumer.
**Preconditions/dependencies:** final T03/T04 bytes and T01 original full revision.
**Paths:** `internal/codexruntime/{manifest.go,shared_history_test.go,rendered_workflow_upgrade_test.go,output_separation_test.go,workflow_skill_content_test.go}`;
inspect `internal/codexruntime/integration.go` and `internal/runtimebootstrap/bootstrap.go`.
**Work/limits:** when any embedded skill changes, append complete replaced four-skill
revision to `sharedSkillHistory`, retaining all historical bytes/digests/receipts and
frozen runtime-only history. Pin final current manifest in `publishedSharedRevisions`.
Review all four skills, including unchanged ones. Retain SkillSetVersion and
BinaryCompatibility 3 unless executable evidence establishes actual incompatibility;
record any local choice, never bump reflexively or redesign install/bootstrap.
**Done/negative:** both temporary Codex/Claude roots converge from previous full set
to final embedded set; repeat install unchanged, modified/foreign content refused,
interrupted/partial update not ready, published ownership retained. Exercise existing
pair-restoration precedent and exact-byte/state-preservation assertions; no global install.
Rollback: normal scoped revert/review of 012 changes, retain published ownership
history if released; compatible pair restoration only through separately authorized
supported mechanism. Project/Workflow/Execution/Profile state and foreign files intact.
**Evidence:** E05 complete replaced/current manifest and four-skill review ledger,
both-root upgrade/conflict/interruption/rollback results. Temporary roots ≠ native hosts.

## AXM39-T06 — Synthetic/package verification and security review

**Owner:** executor/reviewer; CLI, codexruntime and runtimebootstrap package owners.
**Preconditions/dependencies:** T02 coverage and T05 final package/history.
**Paths:** all T02–T05 touched files, this directory's future Evidence; inspect full diff.
**Work/commands:** run focused tests, including actual newly chosen 012 test names:

```bash
go test ./internal/cli -run 'Test(OutputSeparation|WorkflowShowSynthetic|Presentation)'
go test ./internal/codexruntime -run 'Test(RenderedWorkflow|OperatorConversational|WorkflowSkill)'
go test ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./cmd/lingo
./scripts/validate-repository.sh .
git diff --check
./scripts/check-sensitive-files.sh --staged .
```

Use scoped redacted Gitleaks before commit; broaden Go coverage only for actual
impact/failure. Review no production DTO/domain/flags/Registry/Execution/admission
change, preserved history and inert hostile labels/diagnostics; publish sanitized
synthetic public inputs, no account/path/secret/raw-log import. Check links and
AC/case ledger. Report optional validator skips; native maintainer validator scenarios
and `scripts/test-native-acceptance.py` are not AC5 conversational Evidence.
**Done/negative:** applicable actual checks PASS, negatives demonstrably rejected,
independent fact/oracle review complete. Unrun checks remain NOT_RUN; queued/cancelled
CI is not PASS. CI read tied to exact pushed head; local tests are not CI or native.
**Evidence:** E06 command/results, full scoped diff review, security/link checks and
actual CI links/status; per-AC synthetic statuses with native gap retained.

## AXM39-T07 — Two native conversations (external authorization boundary)

**Owner:** specifically authorized host operator and native reviewer, Codex and Claude.
**Preconditions/dependencies:** T06 final revision, compatible installed pair read-back
and concrete action-specific authority. This Tasks stage executes none of these actions.
**Paths:** [existing protocol](examples.md#two-native-conversations--future-protocol-not_run);
future `evidence-implementation.md` separate Codex/Claude records. No host-path edits.
**Protocol/limits:** establish authorized host/surface/build outside maintainer checkout.
For authorized read-back use supported `axiom --json runtime codex status`,
`axiom --json runtime claude status`, `axiom --json docs inspect`, then returned
verified paths and `axiom --json skill inspect <domain-skill>`. Record binary provenance,
packaged skill content identity and compatibility; do not call bootstrap Run for inspection.
Absent/modified/foreign/partial/mixed pair holds dependent session; do not repair receipts.
Installed, verified, loaded, authenticated, authorized and observed remain distinct.
An update, real host invocation/inference, credential/auth access, expense or external
effect lacking specific grant is AUTHORIZATION_REQUIRED for that action. Neither
Intake approval nor predecessor Done supplies it. Record actual loaded session revision;
refresh only within host authority. Safe local work may finish while sessions are NOT_RUN.
**Done/negative:** one real Codex and one real Claude conversation, each natural
list/show/validate→requested ID→both Stages/Agent details→Runtime/readiness and graph/text
fallback, per examples protocol. Replay collisions/blockers labelled synthetic within
session; never manufacture state by dispatching. Preserve failed/partial cases, missing
rendering and untested claims. No inference/E2E duplication or acceptance of AXM-12.
**Evidence:** E07-Codex/E07-Claude: date, host/build, installed/loaded revision,
authorization ref, canonical input/revision, case/AC, sanitized visible excerpts,
PASS/FAIL/NOT_RUN, gaps and next action. No raw transcript/private host inventory.

## AXM39-T08 — AC reconciliation and review handoff

**Owner:** executor/Orchestrator with reviewer; documentation within Spec 012 only.
**Preconditions/dependencies:** T06 for partial report; T07 for complete AC5 Evidence.
**Paths:** future `evidence-implementation.md`, this Tasks status, pertinent 012 index
links and same PR/AXM event. Preserve Spec/Plan and predecessor historical records.
**Work/limits:** join E01–E07 against crosswalk below, exact final PR/SHA and actual
checks. Record technical_decisions[] with rationale and source links, limitations,
remaining_work and next_gate. Publish one matching completion/partial/blocked root
event with structured JSON and raw read-back; deduplicate by attempt/type/SHA.
**Done/negative:** no AC5 completion without both native conversations; synthetic
PASS cannot fill native gaps. Scope review supports AC6 only to actual diff examined.
Show DF-01 prominently at pre-merge, still deferred. No auto-approval, merge/release,
Done, relations/status updates or human acceptance. Real callback may hand off to
review/readiness; unavailable callback means AWAITING_EXECUTION_REPORT, no fake dispatch.
**Evidence:** E08 AC ledger/receipt and same-AXM event URL; incomplete native Evidence
means explicit remaining work and partial implementation handoff, not product completion.

## Crosswalk: unchanged AC ⇄ Tasks ⇄ C01–C16 ⇄ Evidence

| AC / rules | Tasks | Cases | Future Evidence / current product result |
| --- | --- | --- | --- |
| AC1 / P1–P2, L1 | T01, T02, T03, T05, T06, T07, T08 | C01, C02, C03, C04, C05, C06, C16 | E02/E03 exact identity + E05/E06 + E07/E08; NOT_RUN |
| AC2 / S1–S2 | T02, T04, T05, T06, T07, T08 | C07, C08, C15, C16 | E02/E04 summary/detail + graph/text; E05–E08; NOT_RUN |
| AC3 / R1–R2 | T02, T04, T06, T07, T08 | C09, C10, C13, C14 | E02/E04 source/certainty + E06–E08; NOT_RUN |
| AC4 / R3 | T02, T03, T04, T06, T07, T08 | C11, C12, C13 | E02/E03/E04 blockers/effects + E06–E08; NOT_RUN |
| AC5 / L1 | T01–T08 | C01, C02, C03, C04, C05, C06, C07, C08, C09, C10, C11, C12, C13, C14, C15, C16 | E02 regressions + E06 checks + separate E07-Codex/Claude + E08; NOT_RUN |
| AC6 / scope/P2/S2/R2 | T01–T06, T08; T07 authority limits | C01–C16 scope review; C05, C09, C12, C14, C15 explicit negatives | E01/E05/E06/E08 unchanged contracts/history and no certification; implementation NOT_RUN |

## Technical choices, deferred finding and current handoff

TECHNICAL_LOCAL; executor-ai; scope_impact none. Full observations/checks in
[Tasks Evidence](evidence-tasks.md). Each source path below is versioned in this PR.

| ID | Choice / alternative | Rationale / GitHub source evidence |
| --- | --- | --- |
| TASKS-TD-01 | `tasks.md` r1, AXM39-T01–T08; not new issue/runner | Existing [011 Tasks](../011-rendered-workflow-presentation/tasks.md) convention; Plan §5 has eight coherent boundaries |
| TASKS-TD-02 | Direct serial T03→T04; no delegation/concurrent shared writer | Shared [integration](../../../internal/codexruntime/integration.go); hierarchy unproven, Project also owns readiness |
| TASKS-TD-03 | Separate E07 records and partial T08 join; no synthetic substitution | [Existing native protocol](examples.md) and AC5; external authority cannot block independent documentary completion or be invented |
| TASKS-TD-04 | Explicit full-set history/current-manifest pin in T05 | [Published ownership assertions](../../../internal/codexruntime/shared_history_test.go); updating only changed skill digests would lose revision ownership |

DF-01: **DEFERRED_NOT_IMPLEMENTED** — stale predecessor summaries in
[Specifications index](../README.md); provenance [Specify Evidence](evidence-specify.md).
No new requirement; no predecessor correction. Tasks documentary handoff:
`readiness_for_implement`; implementation, Go behavioral verification, installation,
native sessions and acceptance remain future. No demonstrated authorized continuation
callback: `AWAITING_EXECUTION_REPORT`, without ceremonial human continue request.
