# S1-08 correction and recovery — AXM-42

Status: see [Claude recovery (final)](#claude-recovery-s1-08-claude-recovery);
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
