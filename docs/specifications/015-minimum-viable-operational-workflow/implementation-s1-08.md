# S1-08 correction and recovery — AXM-42

Current correction: [DMH recovery attempt 2](#dmh-reconciliation-codex-attempt-2) is **PARTIAL / HUMAN_DECISION_REQUIRED:ARCHITECTURE**.

Historical status: see [Claude recovery (final)](#claude-recovery-s1-08-claude-recovery);
it supersedes the Codex status, matrix and measurements below, which are kept
as history. No acceptance, merge, release or cutover is claimed.
Existing [PR #366](https://github.com/rgomids/axiom/pull/366), branch
`spec/axm-42-minimum-viable-workflow`; authorized S1.1–S1.8 only by
[human authority](https://linear.app/rgomids/issue/AXM-42#comment-32f02fa5).
Implementation runtimes: Codex gpt-6.1-sol (commits `b24fe33`–`1b407a9`,
interrupted by usage quota before this receipt was committed), then Claude
Opus 5.5 medium (`claude-opus-5-5`) recovery attempt `S1-08-claude-recovery`.

## Codex S1-08 draft (historical, uncommitted at interruption)

## Recovery evidence

Before source/index mutation, realpath, owner (`grokbot`, 0700), registered
worktree, origin (`https://github.com/rgomids/axiom.git`), branch and live remote
were checked. Local, remote branch and open PR head agreed at
`bb4754750816b204e02fba9048d905cc0b4bb214`. No Git locks or other visible agent
writers were observed; this is a process-list observation, not a machine-wide
guarantee. Independent status count: 27 entries (seven staged additions, 16
unstaged modified/deleted paths, four untracked entries); no rename. Ignored
`scripts/__pycache__/` remained untouched. The inherited `.s1-03-lingo-test.log`
is absent; no private log was deleted or published.

Private recovery reference: `/home/grokbot/axiom-s1-08-recovery-b6xb74s8/manifest.json`,
SHA-256 `95ccc0d04177b40db9de765be99b19e5564ed4f2babe2a62a8d141b5f297799f`.
Directory mode 0700, patches/objects/manifest 0600. It retains separate staged
and unstaged binary patches, all 27 changed/untracked file contents/modes/hashes,
deletion and symlink metadata. Ignored files were excluded and untouched.
Restoration was verified by applying both patches in an isolated repository and
comparing every manifest object to restored content. The initial absolute-path
`git apply --directory` check refused safely; applying from the isolated
repository succeeded. Snapshot bytes and raw logs remain private and unpushed.

| Recovered paths | Decision and evidence |
| --- | --- |
| `cmd/lingo/main.go`, `runtime_policy.go`, `stage_service.go`, `workflow_binding.go` | KEEP dead-helper deletion; original U1000 and declaration-only call inventory |
| `runtime_policy_fixture_test.go`, deleted `skill_protocol_test.go`, `stage_plan_policy_fixture_test.go`, `workflow_binding_test.go` | KEEP removal of helper functions with no tests/callers; actual installed/CLI tests remain |
| `internal/cli/cli.go` | KEEP unused private request fields removal; canonical parser remains |
| `internal/local/delivery_decision.go`, `delivery_rework.go`, `workflow_store.go`, deleted `workflow_configured_test.go` | KEEP orphan private helpers removal; historical classification/S3 support retained |
| `internal/stageexecution/engine.go` | KEEP cancel poller stop/join so store reads cannot outlive attempt |
| `internal/stageexecution/runtime_launcher.go`, `runtime_launcher_test.go` | FIX fast-command identity gate: require exact token; EOF/publication refusal cannot execute argv |
| Seven `internal/workflowcompiler/testdata/issue306` files | KEEP inert synthetic fixtures, each byte-identical to `05c5787`; legitimate historical decode/inventory consumers |
| `implementation-s1-07.md`, `deletion-s1-07.json` | FIX incomplete receipt; retain historical deletion report as supplementary, current remeasurement supersedes it |
| `.s107chk/`, untracked `lingo` binary | DROP FROM DELIVERY only; move unchanged into private snapshot quarantine; no source discarded |

Commit plan: orphan cleanup/fixtures; canonical safety/provenance corrections;
remaining dead owner retirement and historical fixture corrections; reconciled
Evidence/docs. Scoped staging and sensitive-file review precede each commit.
No reset/clean/stash/rebase/forced checkout/force push or actual cutover.

## Requirement-to-proof matrix

| Task | Status during final validation | Requirement → evidence |
| --- | --- | --- |
| S1.1 | DONE | ADR-0037 accepted decision, exact partial supersessions/backlinks/index; ADR governance structural check PASS (38 ADRs) |
| S1.2 | DONE | Execution4 codec and `executions/v2`; strict/legacy/path refusal, concurrent CAS/history and crash-publication tests in execution_test.go/execution_store_test.go |
| S1.3 | DONE | Frozen selector precedence/Profile optional/default/conflict/refusal tests, no canonical run preflight; policy prediction retirement in S1-08 |
| S1.4 | DONE | Canonical engine crash windows, stale generation, cancel/timeout/PID reuse, unknown/retry/no replay tests; command identity gate negative |
| S1.5 | DONE | Public run minimal/two-stage fixtures, result-bound human review and child refusal, optional commands/output, source provenance, exact output validators, before/after gate and bounded mapper tests |
| S1.6 | DONE | `TestTwoStubQuestionAnswerIngress` plus engine non-running-attempt ingress negatives; S2 transport/parallel execution not claimed |
| S1.7 | INCOMPLETE pending final C/K checks | No legacy Graph/Single operational wiring; orphan publication/scheduler/stage-outcome/planning owners removed; residual policy prediction removal under final validation |
| S1.8 | INCOMPLETE pending S1.7/final checks | Eleven isolated Cutover tests PASS; installed/offline docs PASS with qualification limits; no actual switch |

Observed before final C/K correction: build/vet PASS, curated staticcheck PASS
on `a85ce30` source (use full SHA from commit list), repository validator PASS
on that source plus local Evidence draft; docs/offline PASS on the recorded
dirty source with explicit native/network/source-denial NOT_RUN. Exact final
source/check/CI results supersede these intermediate checks below.

## U1000 correction

At original `bb4754750816b204e02fba9048d905cc0b4bb214`, GitHub
[Curated static analysis](https://github.com/rgomids/axiom/actions/runs/41269447840/job/124739003556)
reported 29 U1000 diagnostics, independently read back. No suppression or
checker weakening was introduced. All removed test files contained only
unused helpers, not `Test` functions.

| Symbols | Resolution / consumer proof |
| --- | --- |
| `workflowReferences`, `runtimeObservationSources.scoped`, `lifecycleService.runtimePolicyFailure`, `lifecycleService.stagePreview`, `readStageResult` | Remove private declaration-only helpers after legacy caller retirement; canonical run supplies owned inputs |
| `selectBuiltinForTest`, `workItemRunStartProtocol`, `expandProtocol`, `installStagePlanPolicy`, `selectBuiltinWorkflowExecutable` | Remove test helpers whose subjects/callers were retired; canonical public-entry and installed docs tests retained |
| requestInput `actor`, `stageResultFile`, `automatic`, `gate`, `outcome`, `reference`, `next`, `fact`, `runtime`, `role`, `complexity`, `capabilities`, `runtimePreview`, `expectedRevision`, `active` | Remove unused private parser fields; canonical workflow/stage input types retain active flags |
| `WorkflowStore.saveDeliveryProjection`, `WorkflowStore.retainDeliveryFile`, `validConfiguredExecution`, `validExecutionAddress` | Remove orphan private writers/helpers; retain actual S3/cutover consumers and fencing/CAS tests |

## Retired consumers and measured scope

[Current production lexical inventory](consumers-s1-08.json) records source
lines and exact patterns, not a semantic reachability certificate. GraphService,
NewGraphService, NewScheduler, DispatchReady, RecordStageOutcome, LoadStageOutcomes
and StagePlanningContext have zero remaining declarations/callers. Graph/Single
stores, stage receipt/cancel/dispatch stores, workflowcompiler and legacy CLI
composition had already been deleted by S1-07.

The final residual audit found two mandatory family C gaps after that inventory:
predictive policy-edit readiness and the active Profile-preview selection owner.
S1-08 removes EvaluatePolicy/CheckPolicy/availability and observation wrappers,
returns explicit `runtime_policy_preview_retired`, and reports only
`selected_at_dispatch` for policy edits while preserving dependency/CAS tokens.
Optional auth diagnostics remain separate from run and report `diagnostic_only`.
Setup ReadinessCandidates enumerates explicitly configured inventory; it does
not rank candidates or certify a future run.

Intentional residuals: `executiongraph.OSProcessRunner` and process identity
primitives are canonical dependencies; historical graph/single/outcome codecs
serve inert inventory fixtures; gitworkspace/integration/acceptance support is
S2/S3-owned, with no old store or scheduler attached. GraphAttemptStore and
DispatchRecord remain only as integration/acceptance contracts. WorkflowStore
historical classifiers and delivery packet/publication/decision types remain
for S3; Service.Start is still required by PreviewDeliveryRework. They are not
the canonical operational start/status engine. No new S2/S3 feature was added.

[Tree-wide deletion remeasurement](deletion-s1-08.json) uses baseline
`0a65d9a90e907fe1904241a82878e0196418e351`, all production/test Go files under
`cmd/` and `internal/`, including replacements anywhere. Physical lines include
comments/blanks; exported counts are lexical. Current delta: production
**-24 files / -6,575 lines / -74 exported types / -121 exported funcs/methods**;
tests **-36 files / -10,596 lines / -256 exported funcs**. No S1 success is
inferred from deletion quantity; S3 budgets are not S1 acceptance gates.

## technical_decisions[]

- TECHNICAL_LOCAL: private object manifest and separate patches instead of stash
  or index reconstruction. Costs disk; verified restoration preserves the
  original staged/unstaged distinction. Rollback restores reviewed paths only.
- TECHNICAL_LOCAL: delete declaration-only legacy helpers instead of dummy
  references or suppressions. Original CI diagnostics plus caller inventory
  establish deadness; installed and canonical behavior tests remain.
- TECHNICAL_LOCAL: retain typed source input provenance beside ArtifactRef,
  rather than changing every artifact/result reference. Workflow digest/CAS
  freeze it; missing/duplicate/conflicting source metadata refuses.
- TECHNICAL_LOCAL: fixed shell gate with positional argv and exact stdin token
  instead of racing direct exec identity observation. EOF closes without exec;
  durable publication releases the token. Same process identity/group survives
  exec. Literal-argv, callback refusal, EOF and cancel tests cover this choice.
- TECHNICAL_LOCAL: use CI Go 1.26.0 and pinned Staticcheck v0.8.1; installed
  Go 1.27.3 export data is unsupported by that checker. Unset inherited GOROOT
  for the script, which intentionally sets GOTOOLCHAIN=local. No checker changes.
- Delegation: independent S1 review/provenance correction, legacy retirement,
  and read-only cutover validation used inherited model/effort, disjoint files
  and bounded outputs. Main session integrates and owns final claims.

## Restrictions and rollback boundary

All execution evidence is deterministic Go tests and isolated local stubs.
Vendor-real E2E, Provider effects, actual private state inventory/backup/stop,
cutover/migration, merge, release, deploy, closing AXM and Product Accepted are
NOT_RUN and not implied. No vendor CLI test invocation was needed. PR metadata
was not changed. Source recovery never reverses process/workspace/remote effects.
Before switching real state, separately authorized operators must stop new
writers, inventory active/unknown work, verify private complete backup and
resolve exact process/effect disposition or BLOCK. After new effects exist,
preserve both namespaces; do not run an old binary against new state, resume
old attempts or replay unknown effects. Prefer a forward fix.

## Claude recovery (S1-08-claude-recovery)

### Recovery verification (before any mutation)

- Worktree `~/axiom-wt/axm-42` (realpath verified), origin
  `https://github.com/rgomids/axiom.git`, branch
  `spec/axm-42-minimum-viable-workflow`. Local HEAD, fetched remote and PR #366
  head all `1b407a9e0605f0d76e7d1ca298a34c85d1d0274f`; CI at that SHA read back
  as PR Checks PASS and delivery-metadata PASS.
- Dirty state counted independently: 25 paths, nothing staged, 19 modified and
  1 deleted (`internal/workflow/list_test.go`) tracked paths, 5 untracked
  Evidence files; only ignored path `scripts/__pycache__/` (untouched). No
  symlinks among untracked paths. Process list showed no other agent or Go
  writer; this is an observation, not a machine-wide guarantee.
- Private snapshot (bytes never pushed or committed): local-only refs in the
  repository object store, outside the worktree.
  `refs/axm42-recovery/s1-08-claude/tracked` → stash-format commit
  `44d9e431c39ffd22c49c5dcccca1f241e07677ff` (parent `1b407a9`) holding the
  exact tracked working tree including the deletion; five
  `refs/axm42-recovery/s1-08-claude/untracked/*` refs pin the untracked
  original blobs by content hash. Integrity: `git diff <ref>` against the live
  tree was empty and the deleted path was absent from the snapshot tree.
  Restore: `git checkout <ref> -- <paths>` / `git cat-file blob <ref>`. The
  sandbox denied writes outside the worktree, so no 0600 file copy was made and
  file modes of untracked files were not separately recorded (all regular
  non-executable documents). The earlier Codex snapshot it references is not
  re-verified here.

### KEEP / FIX / DROP

| Paths | Decision | Rationale / evidence |
| --- | --- | --- |
| `internal/workflow/service.go` (Status, Resume, List, PrepareProjection, Project, AdvanceAutomatic in `gates.go`, `WithProfileContract` in `configured.go`, projection writers) | KEEP | Families E/F/G. No production caller: production composes `workflow.Service` with a nil store and calls only `ResolveLinked`; `NewWorkflowStore` has no production caller |
| `internal/workflow/service.go` deletion of `ProjectionError`/`ProjectionErrorKind` | FIX (restored) | Interrupted edit did not compile; `internal/githubissues/adapter.go` still returns these types |
| `cmd/lingo/main.go` | KEEP | `workflow.New` signature without projection port; build PASS |
| `internal/executiongraph/planner.go` (Planner, ApprovedPlan, WorkUnit, CapabilityValidator) | KEEP | Family B/G operational Plan producer; Proposal remains an inert historical codec |
| `internal/executiongraph/*_test.go` | KEEP | Cycle, unknown dependency, orphan, injected key and unsafe overlap negatives ported onto the historical topology validator |
| `internal/workflow/*_test.go`, deleted `list_test.go`, `local/s8_stores_test.go`, `gitworkspace/git_test.go` | KEEP | Tests of removed APIs only; surviving projection preview/reconciliation assertions call the retained helpers directly. Canonical listing is covered by `internal/local/execution_list_test.go` |
| `implementation-s1-07.md`, `deletion-s1-07.json`, `consumers-s1-08.json` | KEEP | Historical receipts; `consumers-s1-08.json` remains the lexical inventory at `1b407a9` |
| `deletion-s1-08.json`, this file | FIX | Remeasured at `1dba19e`; Codex draft status superseded by this section |

Nothing was dropped. A gitignored scratch measurement program
(`tmp/_axm42measure/`, excluded from `./...`) was created and is not delivered.

### Commits

- `1dba19e840517c9d063282ff8f6ce6cf8a907698`: refactor(workflow): retire
  orphan projection, resume and Plan producer owners (pushed normally after
  verifying that remote `1b407a9` was an ancestor).
- Evidence commit containing this file (its SHA is reported in the stage
  receipt; a commit cannot contain its own hash).

### Retired-symbol and residual proof

At `1dba19e`, `git grep -wE` over `cmd/` and `internal/` for
`AdvanceAutomatic|PrepareProjection|WithProfileContract|profileContract|ProjectionCapability|NewPlanner|ApprovedPlan|WorkUnit|CapabilityValidator|ErrInvalidPlan|ErrUnresolvable|reconcileProjectionRecord|confirmProjectionEffect|upsertProjection|projectionFailure`
returns no match. Earlier families stay at zero per `consumers-s1-08.json`.

Intentionally retained, none constructed by production composition:
`workflow.Service.Start/Transition/RecordLifecycleFact/PreviewDeliveryRework/ValidateTarget`
and `local.WorkflowStore` (S3 family I delivery/rework fixtures and historical
classification); projection preview/validation helpers (historical-state
validation and `reconciliation.go`); `githubissues.Adapter.Inspect/Apply`
(now without an Execution caller, S3.2/S3.4); `executiongraph` Proposal,
process identity and integration contracts (canonical/S2 dependencies);
historical specs that name retired APIs (history preserved, not edited).

### Deletion delta (tree-wide, baseline `0a65d9a`)

[deletion-s1-08.json](deletion-s1-08.json) at `1dba19e`: production
**-24 files / -7,076 lines / -81 exported types / -131 exported funcs/methods**
(330→306 files, 72,716→65,640 lines); tests **-37 files / -11,255 lines /
-286 exported funcs**. This recovery commit alone: production -501 lines /
-7 types / -10 funcs; tests -1 file / -659 lines / -30 funcs. Deletion size is
not an S1 gate; the S3.4 budget is not asserted.

### Validation (source `1dba19e` unless stated)

| Check | Result |
| --- | --- |
| Dirty tree as recovered: `go build ./...` | FAIL (undefined `ProjectionError`), fixed before commit |
| `go build ./...`; `go vet ./internal/... ./cmd/...` | PASS (local, Go 1.27.3) |
| gofmt (`go run cmd/gofmt -l cmd internal scripts`) | PASS (no files) |
| `go test ./... -count=1 -timeout=30m` | PASS (no FAIL; local) |
| Focused `go test -count=1` workflow, executiongraph, githubissues, local, gitworkspace, cmd/lingo | PASS |
| Staticcheck curated checks, local | PASS with v0.8.2 (0 diagnostics); pinned v0.8.1 cannot read Go 1.27 export data locally |
| CI [PR Checks](https://github.com/rgomids/axiom/actions/runs/41305437609/job/124844751050): format, tidy, mod verify, vet, pinned Staticcheck v0.8.1, quality/delivery contracts, build, tests, installed docs journey, repository and sensitive-file checks | PASS |
| CI [delivery-metadata](https://github.com/rgomids/axiom/actions/runs/41305437628/job/124844751239) | PASS |
| `git diff --check` | PASS |
| Local `scripts/*.sh` validators (repository, docs-offline, quality script) | NOT_RUN locally: sandbox allows only git/gh/go; the same scripts PASS in CI above |
| `go test -race` | NOT_RUN: cgo disabled in this environment |
| gitleaks | NOT_RUN: unavailable in sandbox; CI sensitive-file checks PASS |
| Vendor CLI battery, vendor-real E2E | NOT_RUN, not needed |

### S1 task matrix (final)

| Task | Status | Evidence |
| --- | --- | --- |
| S1.1 | DONE | ADR-0037 and supersessions (S1-01); repository/ADR checks PASS in CI |
| S1.2 | DONE | Execution4 codec/store tests in `internal/workflow`, `internal/local`; full suite PASS |
| S1.3 | DONE | Selector precedence/refusal tests; predictive policy retired in `1b407a9` |
| S1.4 | DONE | Engine crash/stale/cancel/PID-reuse/unknown-effect tests in `internal/stageexecution` |
| S1.5 | DONE | Run fixtures, result-bound human review, fenced argv, provenance and validator tests (Codex `c49ca7e`, `e85e225`); re-verified only by the passing suite and CI, not by a new line-by-line review |
| S1.6 | DONE | `TestTwoStubQuestionAnswerIngress` and ingress negatives |
| S1.7 | DONE | No production construction of legacy stores/services; retired-symbol greps zero; residuals classified above; deletion delta recorded |
| S1.8 | DONE (preparation only) | Eleven isolated cutover tests in `internal/local/cutover_test.go` (active/unknown/incomplete, held lease, interrupted publication, unsafe/unreadable/unexpected files, symlink, forged paths, byte-verified backup, no decode/convert/resume); operator docs reconciled in `278eaa7`, unchanged by this recovery. Actual private inventory/backup/stop/cutover NOT_RUN |

### technical_decisions[]

- TECHNICAL_LOCAL: back up to local-only git refs instead of a 0600 directory,
  because the sandbox denied writes outside the worktree. Same bytes,
  content-addressed and restorable; never pushed. Alternative (stash push)
  rejected because it rewrites the working tree.
- TECHNICAL_LOCAL: restore `ProjectionError` instead of also retiring the
  githubissues projection adapter. That adapter is S3-owned (I family); removing
  it is out of S1 scope.
- TECHNICAL_LOCAL: keep `Service.Transition/RecordLifecycleFact` although
  production never calls them, because S3 delivery tests build fixtures with
  them and S3.3/S3.4 owns that deletion.
- TECHNICAL_LOCAL: remeasure deletion with one lexical method on both sides
  (`go/parser` over git objects); baseline exported funcs read 1,096 rather
  than the Codex draft's 1,093.

### deferred_findings[]

- DEFERRED_NOT_IMPLEMENTED (S3.2/S3.4): `githubissues.Adapter.Inspect/Apply`
  and `workflow.ProjectionError` have no Execution caller after this commit;
  retire or rewire with Execution delivery.
- DEFERRED_NOT_IMPLEMENTED (S3.3/S3.4): `workflow.Service.Start/Transition/
  RecordLifecycleFact/PreviewDeliveryRework` and `local.WorkflowStore` are
  reachable only from tests; delete with family I.
- DEFERRED_NOT_IMPLEMENTED (tooling): pinned Staticcheck v0.8.1 cannot analyse
  Go 1.27 export data on hosts without Go 1.26; CI is authoritative.

### Remaining work, rollback and readiness

S1 technical rows are complete on this branch with green CI at `1dba19e`.
Open items: human engineering review of PR #366 (still a draft), the S3
deferrals above, and the separately authorized real cutover (inventory,
backup, stop). Rollback boundary: revert `1dba19e` and the Evidence commit
(source only; no state, process or remote effects were produced). Review
readiness: ready for human review of S1; not accepted, merged or released.


## DMH reconciliation — Codex attempt 2

2026-11-02, same AXM-42 / PR #366 / existing branch. This section supersedes
only the correction disposition above. Overall **PARTIAL**: DMH-06 requires
Rafael's architecture decision; independent re-review is pending. This is not
S1 approval, Product Accepted, merge, release or installed-state cutover.
Authority: [recovery prompt](https://linear.app/rgomids/issue/AXM-42#comment-6329b1ba),
[original DMH review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5732644991).
Implementation: requested Codex gpt-6.1-sol, medium; no Claude invocation,
subagent, vendor E2E or real installed-state mutation.

### Recovery and partial-work accounting

Host `ubuntu-arm-01`, real worktree `/home/grokbot/axiom-wt/axm-42`, owner
grokbot/mode 0700; origin `https://github.com/rgomids/axiom.git`; branch
`spec/axm-42-minimum-viable-workflow`. Local HEAD, remote branch and PR ref
all equalled `387b5130f3f108aae1e2b13bb735c9f1cb1ec193`; merge-base with
`origin/main` was `440e44a68638e08e901fd3f87f5b15cd395a6365`.
Index was clean, 15 tracked files modified and one untracked admission test.
No index lock or second visible agent/Git writer; visible Codex/tmux processes
were this attempt's ancestors. `rg` was unavailable; subsequent searches used
available tools. Initial search command exited 127 for that reason, with no edit.

Before edits, private snapshot `/home/grokbot/axm42-recovery-uhlwndnl`
(mode 0700, private files) preserved porcelain-v2 status, separate staged and
unstaged binary patches, the original index and all 42 relevant dirty/untracked/
ignored files. Manifest SHA-256:
`98f73677b0819aa15284b929da85ceb86ca0361d8d8554f64862a5b327f3221d`.
No symlinks among those 42; approved tracked Claude skill symlinks were retained.
The file archive was extracted privately and every content hash/type/mode
verified. Applying the unstaged binary patch to a private copy of the index
also reproduced all 15 tracked content hashes. Original staged patch was empty.
Raw files, test logs and private provider payloads are not published.

All inherited source was retained: **30 KEEP, 6 REPAIR, 0 DISCARD** at
zero-context hunk granularity (including the new test as one unit). REPAIR
means completion of valid partial work, never wholesale replacement. Every
hunk was compared with Spec015 normal operation/F5, ADR-0037 decisions 2–5
and Tasks S1.2–S1.5. The table records old-file hunk anchors from the snapshot;
test additions spanning two findings retain both mappings.

| Snapshot path | Old hunk start(s) | Finding | Disposition / reason |
| --- | --- | --- | --- |
| `cmd/lingo/canonical_lifecycle_test.go` | 47 | DMH-07 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_lifecycle.go` | 82 | DMH-07 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_lifecycle.go` | 84 | DMH-07 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 43 | DMH-01 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 259 | DMH-01 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 326 | DMH-01 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 341 | DMH-01 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 665 | DMH-03 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 667 | DMH-03 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 698 | DMH-05 | REPAIR: valid partial path retained; complete validation/fixtures |
| `cmd/lingo/execution_run.go` | 722 | DMH-03 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 907 | DMH-01 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run.go` | 944 | DMH-08 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run_test.go` | 10 | DMH-03 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run_test.go` | 657 | DMH-03/04 | KEEP: accepted scoped correction retained |
| `cmd/lingo/execution_run_test.go` | 830 | DMH-08 | KEEP: accepted scoped correction retained |
| `docs/commands.md` | 7 | DMH-04 | KEEP: accepted scoped correction retained |
| `docs/commands.md` | 55 | DMH-08 | KEEP: accepted scoped correction retained |
| `internal/cli/argument_requirements.go` | 106 | DMH-08 | KEEP: accepted scoped correction retained |
| `internal/cli/workflow_stage.go` | 140 | DMH-08 | KEEP: accepted scoped correction retained |
| `internal/cli/workflow_stage.go` | 194 | DMH-08 | KEEP: accepted scoped correction retained |
| `internal/cli/workflow_stage.go` | 262 | DMH-08 | KEEP: accepted scoped correction retained |
| `internal/local/execution_store.go` | 236 | DMH-02 | KEEP: accepted scoped correction retained |
| `internal/local/execution_store_test.go` | 189 | DMH-02 | KEEP: accepted scoped correction retained |
| `internal/stageexecution/engine.go` | 31 | DMH-01 | KEEP: accepted scoped correction retained |
| `internal/stageexecution/engine.go` | 263 | DMH-01 | KEEP: accepted scoped correction retained |
| `internal/stageexecution/engine_test.go` | 233 | DMH-01 | KEEP: accepted scoped correction retained |
| `internal/workflow/execution_default.go` | 6 | DMH-04 | KEEP: accepted scoped correction retained |
| `internal/workflow/execution_default.go` | 9 | DMH-04 | KEEP: accepted scoped correction retained |
| `internal/workflow/execution_default.go` | 13 | DMH-04 | KEEP: accepted scoped correction retained |
| `internal/workflow/execution_default.go` | 21 | DMH-04 | KEEP: accepted scoped correction retained |
| `internal/workflowdefinition/definition.go` | 131 | DMH-05 | REPAIR: valid partial path retained; complete validation/fixtures |
| `internal/workflowdefinition/validation.go` | 5 | DMH-05 | REPAIR: valid partial path retained; complete validation/fixtures |
| `internal/workflowdefinition/validation.go` | 138 | DMH-05 | REPAIR: valid partial path retained; complete validation/fixtures |
| `internal/workflowdefinition/workflow-v2.schema.json` | 452 | DMH-05 | REPAIR: valid partial path retained; complete validation/fixtures |
| `cmd/lingo/execution_admission_test.go` | entire new file | DMH-01 | REPAIR: valid partial path retained; complete validation/fixtures |

Additional repairs: schema-2 Profile omission must pass Workflow authoring;
shadowed selectors must be validated before precedence; direct fields and
historical schemas require round-trip proof. Added invocation tests, a real
persisted installation rebind between stages and retained-artifact collector
cases. No accepted ADR, ADR index, architecture policy, validator or prior
signal/orphan test was changed.

The reported `c.failure(nil)` panic was **not reproduced in the inherited
snapshot**: its test-only `refusal` helper already returns an empty category
for nil. The production mapper is an error-only API. That guard was retained;
archival tests now explicitly require a non-nil error. Replacing only
`execution_run.go` with baseline bytes produces a normal assertion failure
(`err=nil`, completed, two spawns), not a panic or false PASS. Current tests
prove archival prevents stage-two spawn, pre-spawn archival records a failed
prepared attempt with no process, and unchanged/reactivated Project proceeds.

### Finding-to-proof matrix

Baseline reproductions use `go test -overlay <private-json> <package> -run
<test-regex> -count=1 -timeout=90s`, with only the finding's production file(s)
restored from `387b513` and current regression tests. They never mutate the
worktree. Expected-red runs all exited 1. Targeted current tests exited 0.

| Finding | Old / retained Claude work | Baseline proof → current negative and positive proof | Status | Code commit |
| --- | --- | --- | --- | --- |
| DMH-01 Major | KEEP admission refresh and engine hook; REPAIR test assertions/actual persisted rebind | Baseline archived Project completed with two spawns. `TestExecutionDispatchRechecksProjectAdmissionBetweenStages`, `TestExecutionDispatchRechecksProjectAdmissionBeforeSpawn`, `TestExecutionDispatchRefusesRebindRetarget`, `TestExecutionDispatchRefusesPersistedRebindBetweenStages`, `TestEngineAdmitRefusalBeforeSpawnRecordsNothingRan`: revoked/rebound next dispatch has zero spawns; bounded category; unchanged/reactivated positive runs | CORRECTED; independent review pending | `825554ed284e686bf82aaa14412116e43a73602a` |
| DMH-02 Major | KEEP rooted lease fence and same-epoch two-store regression | Baseline foreign lease Save admitted while A held its lease. `TestExecutionStoreRejectsForeignStateRootLease`: Save and Update refuse foreign B/third-root fences, byte/revision unchanged; A-owned lease succeeds | CORRECTED; independent review pending | `ad86601c5f960488e231f01f0d8fa65c394d25af` |
| DMH-03 Major | KEEP criterion obligation/orphan-producer check and tests; add actual collector proof | Baseline lowered Evidence as optional. `TestMapWorkflowDefinitionPreservesCriterionOutputObligation`, `TestCriterionReferencedOptionalArtifactRequiresValidatedPresence` (absent/invalid/present), `TestPublicWorkflowRunIndependentOptionalOutputStaysOptional`: absent or wrong-attempt retained Evidence blocks successor; exact validated Evidence permits both stages; unrelated optional remains optional | CORRECTED; independent review pending | `825554ed284e686bf82aaa14412116e43a73602a` |
| DMH-04 Major | KEEP mapped Work Item input, default revision r2 and public packet test | Baseline packet Inputs empty. `TestPublicWorkflowRunMinimalDefaultReceivesSelectedWorkItem`: two public stub packets have distinct exact Provider/canonical resource-ID/Repository links; confidential environment marker absent; nonexistent Provider executable ensures no fetch; custom input paths retained | CORRECTED; independent review pending | `825554ed284e686bf82aaa14412116e43a73602a` |
| DMH-05 Major | REPAIR in-progress invocation definition/schema/validation/lowering; complete optional Profile authoring and all-source validation | Baseline mapper fails direct-only and run-over-Workflow cases. `TestPublicWorkflowAgentInvocationPrecedence` (7 cases), `TestPublicWorkflowAgentInvocationRequiresEffort`, `TestSchema2InvocationRoundTripAndInvalidSelectors`, `TestResolveInvocationRejectsEveryShadowedInvalidSelector`, existing `TestPublicWorkflowRunInvalidLocalDefaultsHaveNoExecution`: per-level Profile defaults, direct replacement, run > Workflow > local; missing effort/invalid/shadowed selectors refuse; schema1 digest unchanged and legacy effort refused by schema2 | CORRECTED; independent review pending | `72cf04f557b6bf6e5debecaa2fe78c87996e81e0` |
| DMH-06 Major | No inherited normative hunk | Reproducible documentary assertion below exits 1. ADR governance structural checker passes but cannot prove semantic reconciliation. Accepted ADRs untouched | HUMAN_DECISION_REQUIRED:ARCHITECTURE | NONE; proposal only outside repository |
| DMH-07 Minor | KEEP canonical token inverse and run/list/status/evidence tests | Baseline list loses resource and reports `owner:repo:7` as external ID. `TestCanonicalLifecycleWorkItemSelectorRoundTrip`: original selector succeeds; changed and malformed identities fail exact comparison | CORRECTED; independent review pending | `825554ed284e686bf82aaa14412116e43a73602a` |
| DMH-08 Minor | KEEP both duplicate guards and key-presence rejection | Baseline parser refuses repeated inputs; baseline application also accepts empty-then-conflicting key. `TestPublicWorkflowRunAcceptsRepeatedInputBindings`: two independent sources succeed; duplicate/conflicting/empty/unused source refuses before Execution create; ordinary duplicate flags still refuse; existing field/select parser cases retained | CORRECTED; independent review pending | `825554ed284e686bf82aaa14412116e43a73602a` |

### DMH-06 authority gate

[ADR-0009 decision 8](../../decisions/0009-parent-child-execution-graph.md#child-scope-authority-and-resolution)
requires capability/complexity requirements. [ADR-0037 decisions 4–5](../../decisions/0037-one-canonical-execution-for-new-runs.md#decision)
require explicit invocation and no predictive selection, but its ADR-0009
supersession entry says **only durable child Executions and separate graph
lifecycle**. The [accepted research map](../../research/axm-42-minimum-viable-workflow-investigation.md#exact-amendment-implications-proposed-history-preserved)
explicitly maps predictive selection removal too. Under [ADR-0018](../../decisions/0018-adr-evolution-and-supersession-governance.md#decision),
structure passing does not resolve this missing fragment annotation/backlink.
No acceptance is inferred from that research link or CI.

Two bounded options for Rafael:

1. **Recommended:** explicitly authorize the exact research-map reconciliation:
   preserve decision 8 historically, add its scoped ADR-0037 supersession
   annotation, expand ADR-0037's backlink; confirm the index already names the
   same partial superseder. No selection behavior change. Lowest artifact cost;
   explicit acceptance history must identify the scope.
2. Require a separate focused ADR acceptance for the same new-run explicit
   invocation rule, followed by exact decision-8 annotation, reciprocal link
   and index entry. More durable artifacts, but a distinct acceptance record.
   Identifier/date remain unallocated until human decision.

Specific proposed text/diffs (unapplied, uncommitted) are private files
`/home/grokbot/axm42-recovery-uhlwndnl/dmh06-options.md`,
`dmh06-option-a.patch`, `dmh06-option-b.patch`. Neither option changes code
here. The authority/supersession history becomes harder to reverse; all
independent corrections continue. No normative ADR/index/policy edit is part
of these commits.

Reproduce the pending documentary gate (exit 1 expected):

```python
from pathlib import Path
old = Path('docs/decisions/0009-parent-child-execution-graph.md').read_text()
new = Path('docs/decisions/0037-one-canonical-execution-for-new-runs.md').read_text()
clause = old.split('8. Runtime/Model Profile resolution', 1)[1].split('9. Credentials', 1)[0]
backlink = new.split('## Supersedes', 1)[1].split('- [ADR-0020', 1)[0]
assert 'capability and complexity requirements' in clause
assert 'capability/complexity' in backlink or 'Superseded by' in clause
```

### Ignored test artifacts and validation

Before cleanup, repository validator exited 1 for exactly
`tmp/cr-s1-07/{mutant-b,after,before,mutant-a}.log`. Each was a regular
mode-0600 grokbot-owned test-generated log, no symlink, matching the snapshot
SHA-256. Contents were classified privately as CR-S1-07 mutation/process test
output. No visible open handles; other-user `/proc` directories were
inaccessible, so no machine-wide absence claim. Only those four files were
unlinked after verified backup; no other data deleted.

Initial build/vet exited 1 because inherited ignored `tmp/dmh-baseline`
contained `engine.go` (stageexecution) and `execution_run.go` (main).
After checksum/handle checks, the entire directory was renamed intact to
`tmp/_dmh-baseline`, excluded by Go's standard underscore rule. Its three
files remain byte-identical and recoverable; `overlay.json` is inert history.
No source was discarded or validator relaxed. Rollback: rename it back; that
also restores the known build obstruction.

Validation source is the complete code tree delivered by
`825554ed284e686bf82aaa14412116e43a73602a`; earlier invocation HEAD was dirty
with these same source bytes. The containing Evidence commit changes only
this document. Exact final delivery SHA, final-head CI run URLs/conclusions
and publication read-back are recorded in the consolidated top-level PR
reply and the single AXM-42 receipt; they are not inferred from earlier CI.

| Exact command / observation | Result / exit |
| --- | --- |
| Initial `timeout 120s go test ./cmd/lingo ./internal/local ./internal/stageexecution ./internal/workflowdefinition ./internal/cli -count=1 -timeout=90s` | TIMEOUT / 124; no diagnostic output, not PASS |
| Initial narrow inherited DMH regression selection, `go test ./cmd/lingo -run 'TestExecutionDispatch|TestPublicWorkflowRunMinimalDefaultReceivesSelectedWorkItem|TestMapWorkflowDefinitionPreservesCriterion|TestCanonicalLifecycleWorkItemSelectorRoundTrip|TestPublicWorkflowRunAcceptsRepeatedInput' -count=1 -timeout=45s` (outer timeout 60s) | PASS / 0, 9.023s; no panic |
| Final targeted DMH selection across cmd/lingo, local, stageexecution, workflowdefinition, runtimeapplication, cli, projectapp (`-count=1 -timeout=90s`, outer timeout 120s) | PASS / 0; named tests above (cli/projectapp had no matching tests in this selection) |
| `go test ./cmd/lingo -run 'TestCriterionReferencedOptionalArtifactRequiresValidatedPresence|TestExecutionDispatch' -count=1 -timeout=60s` (outer timeout 90s) | PASS / 0, 5.198s |
| `go build ./...` | PASS / 0 after fixture preservation; initial FAIL / 1 separately above |
| `go vet ./...` | PASS / 0 after fixture preservation; initial FAIL / 1 separately above |
| `go test ./... -count=1 -timeout=30m` | PASS / 0, 323.49s including process audit; cmd/lingo 198.810s |
| `gofmt -l cmd internal scripts` | PASS / 0, empty output; formatting touched changed files only |
| `go run ./scripts/check-architecture.go domain` | PASS / 0, 12 files |
| `go run ./scripts/check-architecture.go application` | PASS / 0, 46 files |
| `PYTHONDONTWRITEBYTECODE=1 bash scripts/validate-repository.sh .` | PASS / 0 after four-log cleanup; initial FAIL / 1; includes ADR, bootstrap/skills, automation and sensitive-file checks |
| `./scripts/check-sensitive-files.sh --staged .` | PASS / 0 before each code commit; staged paths/content reviewed |
| `git diff --check 440e44a68638e08e901fd3f87f5b15cd395a6365...HEAD` | PASS / 0, empty output on code delivery; repeat on final publication |
| `git diff 387b5130f3f108aae1e2b13bb735c9f1cb1ec193 -- docs/decisions .agents/policies` | PASS / 0, empty; no normative changes |
| DMH-06 documentary assertion | FAIL / 1, expected unresolved authority gap; never treated as green |
| gitleaks / staticcheck / race / vendor E2E / native Darwin | NOT_RUN: optional binaries absent / race not required / vendor E2E prohibited / unavailable platform |

The full-suite observer sampled only descendants of its own `go test` process,
recording PID/start/group identity before termination and checking the same
identities afterward: **523 observed, zero non-zombie survivors**, no cleanup.
The retained seven-mode orphan test independently checks its own process families
(including SIGKILL, forced failure and test timeout), fails if emergency cleanup
is needed, and passed in this full suite. No process belonging to another run
was signalled. Sampling does not claim exhaustive machine-wide observation.
Prior CR-S1-06/07, IF-01, IR-01..09 and F-01..05 source/tests remain;
regression preservation is established by the full suite, not claimed as
independent re-review. No broad process kill or independent-process cleanup.
The pre-spawn check is a fresh bounded observation, not an atomic filesystem/
process transaction or a guarantee against concurrent same-UID mutation.

Exact final targeted selection (exit 0; only deterministic temporary fixtures):

```bash
timeout 120s go test ./cmd/lingo ./internal/local ./internal/stageexecution ./internal/workflowdefinition ./internal/runtimeapplication ./internal/cli ./internal/projectapp -run 'TestExecutionDispatch|TestPublicWorkflowAgentInvocation|TestSchema2Invocation|TestResolveInvocation|TestExecutionStoreRejectsForeignStateRootLease|TestEngineAdmit|TestMapWorkflowDefinitionPreservesCriterion|TestPublicWorkflowRunIndependentOptional|TestPublicWorkflowRunMinimalDefaultReceives|TestCanonicalLifecycleWorkItemSelectorRoundTrip|TestPublicWorkflowRunAcceptsRepeatedInput' -count=1 -timeout=90s
```

### Technical decisions and remaining work

`technical_decisions[]`:

- TD-DMH-RECOVERY: snapshot every relevant path and copied index before edits;
  retain partial source even when tests fail. Alternative blind replacement
  rejected. Restore exact archive/index/patches only after coordinating writers.
- TD-DMH-ADMISSION: fresh resolve/admission before preparation plus a final
  engine hook after Build; retain frozen workspace/invocation. Alternative
  silent retarget rejected. Rollback only via reviewed code revert.
- TD-DMH-INPUTS: require only criterion-referenced optional outputs; map the
  selected Work Item explicitly, bump minimal revision r2, preserve r1 snapshots;
  repeat existing input flag while rejecting duplicate keys. No new public flag.
- TD-DMH-INVOCATION: finish the inherited schema-2 invocation object and optional
  Profile path; validate every source before precedence. Preserve old document
  digests and schema1 effort semantics. No model capability catalogue or S2/S3.
- TD-DMH-FIXTURES: preserve obstructing ignored overlay directory by reversible
  rename; delete only the four explicitly authorized checksum-backed log copies.

`deferred_findings[]`: DMH-06 is blocking HUMAN_DECISION_REQUIRED:ARCHITECTURE,
not an optional deferred fix. Previously unproven/out-of-scope IF-02 remains
DEFERRED_NOT_IMPLEMENTED at pre-merge (see historical Evidence); no new work
was introduced for it.

`remaining_work`: Rafael's DMH-06 decision and authorized documentary follow-up;
fresh context-independent Claude Opus 5.5 review of **all changed hunks**, the
DMH regression matrix and authority gate, after the reported 05:10 BRT quota
reset and verified availability. Codex authored the original review and these
corrections and **cannot certify independent review**. No review is dispatched
inside this implementation run. No third attempt or automatic next-stage
callback is launched.

`next_gate`: **HUMAN_DECISION_REQUIRED:DMH-06**, then
**INDEPENDENT_CLAUDE_OPUS_5_5_TARGETED_RE_REVIEW** on the exact final HEAD.

## DMH-06 Option A reconciliation — Claude Phase 1

2026-11-02, same AXM-42 / PR #366 / existing branch, base
`739588dd3d9b12dc5a7ff230af3882e973c4e139`. This section supersedes only the
DMH-06 disposition above; the attempt-2 history stays as written. Executor:
Claude Code `claude-opus-5-5`, effort medium. That also makes this session the
author of this commit, so it **cannot certify independent review**. No Product
Accepted, merge, release, vendor E2E or installed-state cutover.

### Decision authority and scope

Rafael decided **Option A — limited supersession reconciliation** for DMH-06
on 2026-11-02 at 10:21 America/Sao_Paulo. The decision was supplied by the
Maestro operator handoff to this session. No separate Linear decision record
was read or verified here. Approved scope: keep ADR-0009 decision 8 as
historical text; add a bounded ADR-0037 supersession annotation after it;
extend ADR-0037's reciprocal backlink; verify the ADR index. No new ADR, no
other accepted ADR changed, no new selection algorithm, no Runtime/model
behavior change. This completes the ADR-0009 row of the already accepted
[research amendment map](../../research/axm-42-minimum-viable-workflow-investigation.md#exact-amendment-implications-proposed-history-preserved)
("Remove predictive capability/complexity resolution").

### Applied change

Private patch `dmh06-option-a.patch`, SHA-256
`5c092fdd17708cfb02194173deeec92b7a4b770cd4ea88822b663710bb17cb7b`
(options note `933336238f5b90e413cb1a4753ae272080ee922bab4d6586f723ce410df3eb28`).
Its whole content was inspected against the scope above, checked with
`git apply --check` and applied unmodified. Original files were backed up
privately first. Total: 2 files, 9 insertions, 1 deletion.

- [ADR-0009](../../decisions/0009-parent-child-execution-graph.md#child-scope-authority-and-resolution):
  insertion only, after decision 8. Decision 8 itself is unchanged:

  > Superseded by [ADR-0037 — One canonical Execution for new workflow runs](../../decisions/0037-one-canonical-execution-for-new-runs.md#decision), accepted 2026-10-30.
  > Only capability/complexity-based Runtime/Model Profile selection for new runs
  > is superseded. Explicit target restrictions, frozen invocation identity and
  > no silent substitution remain. Historical text is preserved for traceability.

- [ADR-0037 Supersedes](../../decisions/0037-one-canonical-execution-for-new-runs.md#supersedes),
  ADR-0009 entry. Before: "only durable child Executions and separate graph
  lifecycle for new runs." After: the same clause ending in `;`, plus
  "and decision 8, only capability/complexity-based Runtime/Model Profile
  selection for new runs. Invocation ownership and explicit target
  restrictions follow decisions 4–5 above."
- [ADR index](../../decisions/README.md): unchanged. The ADR-0009 entry already
  carries `Partially superseded by [ADR-0037](0037-one-canonical-execution-for-new-runs.md)`,
  which is all ADR-0018 §7 requires.

Preserved: ADR-0009 status `Accepted`; decisions 1–7 and 9–24; the
installed/enabled/allowlisted restriction, no-match block and no silent
fallback, now stated as the retained scope; legacy records. The annotation
applies to new canonical runs only.

### Validation

Executed on the working tree at base `739588d` with only this patch applied
(the Go result also holds for the commit, since it changes no Go file):

| Exact command / observation | Result / exit |
| --- | --- |
| `git apply --check -v dmh06-option-a.patch` | PASS / 0, both files |
| `git diff -U0` on ADR-0009 | insertion-only hunk `@@ -85,0 +86,5 @@`; decision 8 lines 81–85 unchanged |
| `git diff --stat 739588d -- docs/decisions .agents/policies` | only ADR-0009 and ADR-0037 |
| Pre-existing DMH-06 documentary assertion (above), evaluated by equivalent `grep -n` because local Python needed approval this session was not granted | PASS: decision-8 slice has the historical text (`:81`) and the annotation (`:87`) before `9. Credentials` (`:91`); ADR-0037 backlink has `capability/complexity` (`:71`) and the `#child-scope-authority-and-resolution` anchor (`:70`) |
| GitHub GFM render (`gh api markdown`) of an annotation directly followed by `9.` | blockquote closes and the list resumes at 9 |
| `go build ./...` | PASS / 0 |
| `go vet ./...` | PASS / 0 |
| `go test ./... -count=1 -timeout=30m` | PASS / 0; every package `ok`, cmd/lingo 180.392s, local 94.300s, workflow 80.472s, stageexecution 20.200s |
| `go run ./scripts/check-architecture.go domain` | PASS / 0, 12 files |
| `go run ./scripts/check-architecture.go application` | PASS / 0, 46 files |
| `git diff --check` (uncommitted patch) | PASS / 0, empty |
| `gofmt -l cmd internal scripts`; `PYTHONDONTWRITEBYTECODE=1 bash scripts/validate-repository.sh .` (includes `check-adr-governance.py`); `ps` process sampling | NOT_RUN locally: command approval was not available in this non-interactive session. No Go file changed. ADR structure and validator are proven only by PR Checks on the new SHA |

Process-identity observation was not available (no `ps`), so this run makes no
zero-survivor claim of its own. The retained orphan/signal tests are part of
the full suite above.

### Technical decisions, deferrals and next gate

`technical_decisions[]`:

- TD-DMH06-VERBATIM: apply the approved patch byte-for-byte. A blank line
  after the annotation (to match precedent) was considered and rejected:
  GFM renders it correctly, and editing it would depart from the approved
  bytes. Rollback: `git revert` of this commit.
- TD-DMH06-INDEX: no index edit. It is structurally consistent, and the options
  note asked for verification without unnecessary edits.

`deferred_findings[]` (DEFERRED_NOT_IMPLEMENTED):

- DF-DMH06-INDEX-TEXT: the ADR-0009 index free text still names only "durable
  child Executions and separate graph lifecycle". It is navigation-only and
  does not mention decision 8. Any update needs its own scoped decision.
- Carried forward unchanged: DF-07-2 vendor SIGKILL NOT_PROVEN, DF-07-3 Darwin
  NOT_RUN, DF-IR-01, D-01/02/04/05, IF-02.

`next_gate`: **FRESH_SESSION_CLAUDE_READONLY_REVIEW**, an independent
read-only review of `387b513...<final SHA>` covering DMH-01..08.
