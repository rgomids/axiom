# S1-04 recovery checkpoint — AXM-42 (PARTIAL)

Attempt `S1-04`, stage `implement`, [existing draft PR #366](https://github.com/rgomids/axiom/pull/366),
branch `spec/axm-42-minimum-viable-workflow`. Initial local HEAD, remote branch
and PR head were independently observed at
`aec00bed9a531e76afc4aae1b09b82d9cbb85ce3`. No Linear access/publication.
Authority is the S1-04 dispatch and its supplied human S1 authorization;
accepted Spec 015, Plan, Tasks and ADR-0037 are unchanged.

This is **partial**, not S1 completion, review readiness or Product Accepted.
No real installed-state inspection, backup, stop or cutover occurred.

## Recovery preflight and preservation

Verified physical cwd and Git top-level are the supplied existing worktree;
branch, origin (`https://github.com/rgomids/axiom.git`), worktree registration,
owner (`grokbot`), directory mode 0700 and `.git` pointer mode 0600 matched.
Process inspection showed this Codex executor and its host; no second executor
or competing Git mutation was observed. `rg` was absent; searches used `grep`
and Python. No fresh clone, branch switch, reset, clean, restore, stash or rebase.

Initial inventory: **94 dirty paths: 56 staged deletions, 29 unstaged modified,
9 untracked**. `git diff --check` reported one trailing blank line in
`cmd/lingo/stage_service.go`, subsequently fixed. Ignored local log and Python
bytecode were inventoried by name and excluded from the source snapshot.

A private directory outside the worktree holds staged/unstaged/HEAD binary
patches, regular source contents, deletion list, modes, sizes and SHA256 hashes;
every included copy was reread and verified. Manifest SHA256:
`4bcc6b5ab4c02be9bb27e3389812dd7da667a6ed81440e5e31c8cab838a17bb4`.
Snapshot was never uploaded or staged. The untracked `lingo` ELF build artifact
matched high-confidence secret patterns, so it was excluded from the snapshot,
never executed and subsequently moved intact into a separate private quarantine.
Its byte hash before/after move is
`e3560c5e735dc536130a6727dc031e1eb0d7dd43669caea10923b1abb9d6d278`.
No claim is made that those patterns were real credentials or harmless literals.

[Path-by-path recovery inventory](recovery-s1-04-inventory.json): **77 KEEP,
16 FIX, 1 DROP**. DROP means the individually verified generated binary was
removed from the source worktree by preservation in quarantine, not deleted.
No inherited source modification was discarded. FIX preserves the attempted
implementation while repairing compile failures, safety controls and metadata.

## Recovered implementation and review

The public `workflow run` now drives the S1-02 RuntimeLauncher and canonical
Execution engine with linked Project/Repository/Work Item identity. Stage status,
cancel, retry and reconcile address that aggregate. Optional Profiles supply
values; per-agent overrides freeze the effective tuple/provenance. Ordinary
retry preserves it; `--new-selection` permits a new attempt without rebinding old
snapshots. Missing outputs fail, lost terminal evidence remains unknown, held
leases refuse and cancellation signals only owned correlated process scope.

Removed Graph/Single stage dispatch, detached supervisors, stage-plan CLI,
Validator Policy store and stagepublication producers; repaired stale test and
operator diagnostic references rather than restoring removed sentinels. Stage
plan/run/integrate return bounded replacement guidance. Historical delivery and
coordination readers **remain**, as do old workflow lifecycle commands/stores
and predictive policy consumers: structural S1.7 is not complete.

Scoped review corrected three inherited shortcomings:

- Writable retries now require explicit `--disposition preserve`, recorded in the
  new attempt; unsupported reset refuses. No automatic Git reset or workspace
  cleanup exists. Previous attempts stay immutable.
- Configured validators and before-gates refuse before Execution creation.
  Substituting a later human gate would silently weaken their semantics.
  Implementing their exact checks remains open.
- Schema-2 definitions permit output-only stages without compulsory validators,
  six SDD phases or all protected SDD gates. Configured gate positions, dangling
  validators, bounded parsing and stage/agent controls still validate; schema 1
  is unchanged. This implements the approved optional-SDD/default-work contract,
  not a new specification or authority grant.

The installed Work Item skill, command metadata/help, history ownership and
code-coupled commands reference are reconciled to the actual partial surface.
A synthetic uncommitted skill-manifest pin inherited from S1-03 was replaced
with the actual embedded digest, not registered as a fictitious publication.
PR metadata has not been changed: provide the suggested update below to Maestro.

## Ported invariant coverage

Retired-only subjects were removed with their implementation. Surviving controls
are exercised at the replacement boundary, including:

| Invariant | Replacement tests |
| --- | --- |
| one writer, revision CAS, stale/released fence | `TestExecutionStoreRevisionCASAndSingleRunner`, `TestExecutionStoreReleasedOrSupersededLeaseCannotCommit`, `TestExecutionStoreConcurrentWritersSerializeWithoutLoss` |
| persist before spawn; unknown never replays | `TestEngineCrashWindowsBecomeUnknownAndNeverRespawn`, `TestPublicWorkflowRunRunnerFenceAndUnknownEffectsNeverReplay` |
| rooted paths, private permissions, no symlink | `TestExecutionStoreRejectsSymlinkedNamespace`, `TestRuntimeLauncherRefusesUnsafeMachineLocalInputs` |
| PID reuse, cancellation, timeout, orphan liveness | `TestEngineReconcileNeverSignalsAReusedPID`, `TestEngineCancelDuringRunStopsOwnedProcessGroup`, `TestEngineTimeoutStopsProcessAndFails`, `TestEngineReconcileReportsLiveOrphanWithoutSignalling` |
| held lease, retry limits, workspace disposition | `TestEngineReconcileRefusedWhileRunnerHeld`, `TestEngineRetryStopsAtAttemptLimit`, `TestEngineWritableRetryRequiresExplicitPreservation` |
| selectors, redaction, missing outputs | RuntimeLauncher suites, `TestEngineRedactsLauncherError`, `TestOutputSeparationSanitizesBeforeBothPresentations`, public run failure/retry tests |
| immutable ordinary/fresh retry; restrictions | `TestEngineRetryPreservesFrozenSelectionAfterLocalDefaultsChange`, `TestEngineExplicitNewSelectionFreezesFreshTupleAsNewAttempt`, domain selection tests |
| public automatic two-stage progression | `TestPublicWorkflowRunTwoStagesProgressWithoutManualIntervention` uses local shell stubs, real command parser/composition/store/launcher, no vendor/Provider |
| real gate and unsupported controls | `TestPublicWorkflowRunSelectedWorkflowStopsAtRealGate`, `TestPublicWorkflowRunRefusesUnsupportedWorkflowControls` |

This is a scoped coverage map, not proof that all remaining A–H/K consumers have
been retired. Git integration/delivery replacement remains S2/S3 work; no lost
remote effect is classified as success.

## S1 task matrix

| Task | Observed status | Remaining obligation |
| --- | --- | --- |
| S1.1 | Previously delivered, preserved | No new ADR acceptance claimed |
| S1.2 | Preserved plus explicit selection/disposition fields | Broad local run passed; final-SHA CI status in receipt |
| S1.3 | Resolver preserved | Public machine-local defaults layer still unwired |
| S1.4 | Preserved plus bounded safety corrections | Review remaining consumers before S1-wide certification |
| S1.5 | PARTIAL | Public run and local two-stage fixture delivered; machine-local defaults and exact configured validator/before-gate execution remain |
| S1.6 | Previously delivered, preserved | No S2 parallel transport claim |
| S1.7 | PARTIAL | 56 obsolete files removed; retained lifecycle/Graph/Single readers, stores, policy/auth consumers still require structural retirement |
| S1.8 | PARTIAL | Isolated backup hardening/negatives and docs reconciliation delivered; final full validation and stop-or-block completeness remain |

## Validation ledger

Commands below are observed local deterministic evidence, not final-head CI.
The first full diagnostic run overlapped source edits and FAILED; it is not
a final-source PASS. A subsequent complete run exited 0; a further full run and
affected-package revalidation distinguish the final source from that diagnostic.

| Command | Observed result |
| --- | --- |
| initial `go test ./cmd/lingo ./internal/stageexecution ./internal/cli ./internal/workflow -run '^$' -count=1` | FAIL: removed sentinels/types and unused imports in cmd/lingo |
| same compile-only command after repair | PASS, four packages |
| `go test ./cmd/lingo -run 'TestPublicWorkflow\|TestOutputSeparation\|TestOperator\|TestPublishRefuses' -count=1 -timeout=10m` | PASS before later disposition/schema additions |
| `go test ./internal/cli ./internal/codexruntime ./internal/stageexecution ./internal/workflowdefinition -count=1 -timeout=10m` | PASS, 6.206s / 28.083s / 22.673s / 2.739s |
| `go test ./cmd/lingo ./internal/workflowdefinition -run 'TestPublicWorkflow\|TestV2OutputOnly\|TestBuiltin' -count=1 -timeout=5m` | PASS, 10.820s / 0.206s |
| `go test ./internal/workflow -run 'TestExplicitNew\|TestAgentRuntime\|TestOrdinaryRetry\|TestExecution' -count=1 -timeout=5m` | PASS, 1.846s |
| `go test ./internal/local -run TestCutover -count=1 -timeout=5m` | PASS, 0.386s; isolated hardening and negatives |
| `go build ./...`; `go vet ./internal/... ./cmd/...` | PASS before final disposition/schema changes; repeat required |
| `python3 scripts/check-adr-governance.py` | PASS, 38 ADRs |
| `python3 scripts/check-automation-registry.py` | initially FAIL on untracked build binary; PASS after private quarantine, 99 surfaces |
| `bash scripts/check-claude-bootstrap.sh --skill-adapters .` | PASS |
| `bash scripts/check-sensitive-files.sh .` | PASS after private quarantine |
| `go run scripts/check-architecture.go domain` | FAIL: preexisting `encoding/hex` in `internal/project/workflow.go` |
| `go run scripts/check-architecture.go application` | FAIL: preexisting `strconv` in `internal/projectapp/active_workflow.go` |
| `git diff --check` | PASS after trailing-blank correction |
| `python3 scripts/test-go-quality.py` | FAIL, 4 of 10 cases require absent local `staticcheck`; no tool installed, CI uses the pinned tool |
| race / gitleaks | NOT_RUN: no gcc / no installed gitleaks |
| live vendor CLI battery | NOT_RUN; all Runtime fixtures are local stubs; no vendor CLI was invoked |
| real cutover, Provider writes, merge, release, Product Accepted | NOT_RUN, not authorized |
| CI at `477a968` | delivery-metadata SUCCESS; PR Checks FAILURE solely at Formatting (`internal/cli/workflow_stage_test.go`); fixed in `b0deb40` |
| CI at `b0deb40` | delivery-metadata SUCCESS; PR Checks FAILURE at Staticcheck: SA4006 unused gate assignments and U1000 unused retired private helpers; corrected in `a11b6be` |
| complete rerun `go test ./... -count=1 -timeout=30m` | PASS, exit 0; cmd/lingo 483.699s, local 279.173s, workflow 217.199s; scoped post-cleanup checks also passed |
| CI at `a11b6be` | PASS: PR Checks and delivery-metadata, read back from PR #366; final documentation SHA remains separate |
| final-SHA CI | final Evidence commit/push followed by remote read-back in receipt |

An initial explicit staging command failed because inherited deletions were already
absent from the index (`pathspec did not match`). The corrected command staged
only the 46 present audited paths while preserving all 56 audited staged deletions;
no reset/restore or worktree overwrite was used.

Intermediate failures corrected: stale mapper operation name, canonical payload
lost by testing through unrelated Project presentation, missing writable-workspace
fixture dispositions, invalid custom Workflow fixture/schema controls, and
installed skill digest/routing drift. The full diagnostic run may contain compiler
failures from overlapping edits; final evidence must distinguish it explicitly.

## technical_decisions[]

```json
[
  {
    "question": "How to preserve interrupted work before edits?",
    "chosen_option": "Private scoped source copies, staged/unstaged/HEAD patches and verified SHA256/mode manifest; quarantine the sensitive-pattern binary separately",
    "rationale": "Retains tracked changes, untracked source and deletion intent without uploading private contents or copying a potentially sensitive binary into the snapshot",
    "evidence_refs": [
      "recovery-s1-04-inventory.json"
    ],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  },
  {
    "question": "How to repair obsolete API-dependent tests?",
    "chosen_option": "Port surviving invariants to Execution/engine/public command boundaries and remove only retired subjects",
    "rationale": "Restoring sentinels or the old supervisor would restore a dual engine; compile and replacement invariant tests provide executable evidence",
    "evidence_refs": [
      "cmd/lingo/output_separation_test.go",
      "internal/stageexecution/engine_ported_test.go"
    ],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  },
  {
    "question": "How should writable retry handle the prior workspace?",
    "chosen_option": "Require and persist explicit preserve disposition; refuse reset",
    "rationale": "No implicit destructive operation or inferred workspace safety; ordinary retry keeps frozen invocation while a separately explicit new selection creates another attempt",
    "evidence_refs": [
      "TestEngineWritableRetryRequiresExplicitPreservation"
    ],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  },
  {
    "question": "How should partially mapped Workflow controls behave?",
    "chosen_option": "Allow schema-2 output-only minimal stages; refuse configured validators and before-gates until their exact implementation exists",
    "rationale": "Matches optional SDD and no mandatory Policy; avoids substituting human approval for an unexecuted validator",
    "evidence_refs": [
      "TestV2OutputOnlyWorkflowNeedsNoValidatorPolicy",
      "TestPublicWorkflowRunRefusesUnsupportedWorkflowControls"
    ],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  },
  {
    "question": "How should raw cutover backups enforce filesystem authority?",
    "chosen_option": "Existing anchored private bounded reader, exclusive creator, strict relative legacy paths and blocking unknown/post-cutover namespaces",
    "rationale": "A hash does not authorize a pathname or certify the bytes behind a link; path-based copying could certify an incomplete inventory",
    "evidence_refs": [
      "internal/local/cutover.go",
      "TestCutoverUnsafeRootsAndUnreadableRecordsBlock",
      "TestCutoverUnknownNestedNamespacesAndUnsafeRootRefuse",
      "TestCutoverBackupRefusesForgedPathsAndChangedSource",
      "TestCutoverBackupVerificationRefusesSymlink"
    ],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  }
]
```

## deferred_findings[] / remaining work

- Architecture checks reproduce baseline violations at the initial HEAD. Other
  existing Project policy readers also use `strconv`. No allowlist weakening or
  unrelated layer refactor was performed. Investigate in the remaining scoped S1
  consumer retirement; a full architecture PASS is not claimed.
- Public configuration still lacks wiring for the canonical local-default layer.
  Do not derive a new selection from capability/complexity preferences.
- Configured validators/before-gates are not implemented exactly and are refused.
  Complete the retained Workflow mapping without skipping controls or inventing
  human approval. Legacy schema-2 maximum-attempt/timeouts and agent dependency
  semantics also need complete reconciliation before broad Workflow claims.
- Active historical lifecycle/coordination/delivery readers and Graph/Single
  stores remain. No claim of zero legacy consumers or complete A–H/K deletion.
- Review cutover completeness under unknown nested entries, process/attempt
  disposition and publication interruption; never run against installed state.
- Complete final full tests, validators, exact final-SHA CI read-back and scoped
  engineering/security review. S1 remains unready for the review gate.

## Suggested PR metadata (not applied)

Title: `feat(workflow): AXM-42 canonical Execution — partial S1 recovery`

Body: `Implements the canonical workflow run path and frozen per-attempt
invocations, retires Graph/Single stage dispatch and supervisors, and preserves
interrupted S1-03 work through a private verified recovery snapshot. S1 remains
partial: local defaults, exact configured controls and remaining legacy consumers
are unresolved. See implementation-s1-04.md for validation, deletion metrics and
cutover limits. No real cutover, live vendor E2E or Product Accepted.`

## Commits / next gate

Commits already created and pushed normally to the same draft PR:

- `477a968d17acea3c859763621c3b506eb12a864c`: recovered public CLI, retired
  dispatch/supervisors, ported tests, schema-2 minimal definitions and docs.
- `556901631d9c46139deb4dd84cdde7542b709715`: isolated cutover inventory/backup
  safety and deletion report.
- `b0deb4011419c4397e1422c8ed05ce17dd4a2258`: formatting-only repair of inherited
  parser fixture, after observed CI formatting failure.
- `a11b6be2272c505af7dc025aaadfd5f00687191f`: remove three obsolete private fixture/diagnostic
  helpers/field and two unused assignments reported by Staticcheck.

Before each push, fetched the branch, read PR HEAD and checked remote ancestry;
remote heads were `aec00bed...`, `477a968...` and `b0deb40...`, all compatible.
The final Evidence-only commit cannot embed its own SHA; the receipt records it. No new PR, force push, merge or Linear publication.

Next gate: `resume_implement`. `linear_published:false`. **REPORT_PENDING**:
Maestro publishes the final same-AXM-42 receipt and API-reads it back; no callback
or autonomous monitoring is claimed.

## Cutover hardening and deletion measurement

The cutover helpers operate only on isolated fixtures in this attempt. They now
open owned anchored directories and bounded private regular files through the
existing no-symlink reader; reject untrusted inventory traversal/absolute paths;
verify both source size and hash; create backup files exclusively; re-read actual
backup bytes; reject linked backups, unreadable files/directories and unsafe root
permissions. Unknown operational versions and any post-cutover Execution namespace
block a pre-switch inventory instead of disappearing from it. Active/held lease,
interrupted publication, source-byte drift and missing human disposition remain
blocking. No source conversion, legacy resume, signal, deletion or real cutover.
The per-record backup/inventory bound is the existing `MaxExecutionBytes` (4 MiB);
oversized records block rather than being silently omitted.

`go test ./internal/local -run 'TestCutover|TestExecutionStore' -count=1
-timeout=5m`: PASS, 3.304s after all cutover production changes.
Final `go build ./...` and `go vet ./internal/... ./cmd/...`: PASS after those
production changes. `bash scripts/check-go-quality.sh format`: PASS after the
format-only correction. No new automation script or registry surface was added.

[Deletion report](deletion-s1-04.json), exact baseline candidate union plus every
added Go file under `internal/` and `cmd/` since `0a65d9a...`:

| Metric | Approved baseline | S1-02 HEAD | S1-04 current | Total baseline delta | Recovery delta from S1-02 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Production files | 88 | 101 | 79 | -9 | -22 |
| Production physical lines | 21,872 | 24,578 | 20,055 | -1,817 | -4,523 |
| Production exported types | 287 | 336 | 251 | -36 | -85 |
| Production exported funcs/methods | 351 | 398 | 317 | -34 | -81 |
| Test files | 84 | 92 | 75 | -9 | -17 |
| Test physical lines | 18,320 | 20,126 | 14,858 | -3,462 | -5,268 |
| Test exported functions | 481 | 534 | 390 | -91 | -144 |

These are physical lines/declarations, not semantic reachability. The 56-file
inherited retirement and replacements are included, so relocation cannot masquerade
as deletion. **S3's net -15 production files/-4,000 lines/-40 types is not met and
is not an S1 gate**. S1.7 still requires actual producer/consumer retirement.

Observed retained consumers, not a zero-legacy claim: `delivery.go:65,121`,
`stage_publish.go:37,101,113`, `stage_service.go:41,50,162,195`, and
`runtime_policy.go:145`. `git grep` finds Graph/Single lookups/store construction
and the predictive diagnostic mapping there. Historical lifecycle commands remain
in `internal/cli/commands.go`; no compatibility engine was restored to compile.

Additional technical decision: use the existing anchored private reader and
exclusive creator for raw backups, and block unknown/post-cutover namespaces.
Chosen over path-based copying because a digest alone does not authorize a path
or certify a symlink's referent. `scope_impact:none`,
`decision_maker:executor-ai`; evidence is the cutover negative suite.

Post-CI cleanup validation: `go test ./cmd/lingo ./internal/stageexecution -run
'TestPublicWorkflow|TestOutputSeparation|TestOperator|TestEngine' -count=1
-timeout=5m`: PASS, 24.691s / 30.913s. `go build ./...`,
`go vet ./internal/... ./cmd/...` and `bash scripts/check-go-quality.sh format`:
PASS afterward. The removed helpers had no callers; no invariant test was removed
by this cleanup and no static-analysis exclusion was added. Local Staticcheck is
NOT_RUN (not installed); pinned CI Staticcheck supplies that evidence.

The complete local rerun exited 0 after the initial diagnostic failure:
`cmd/lingo` 483.699s, `internal/local` 279.173s,
`internal/workflow` 217.199s. A second complete run started at source
`b0deb4011419c4397e1422c8ed05ce17dd4a2258`; only the documented unused-code cleanup
and Evidence changes followed. Local results are composite source evidence,
not a claim that a pristine final documentation commit ran all tests atomically.
CI at `a11b6be` completed SUCCESS for PR Checks and delivery-metadata.
[PR Checks](https://github.com/rgomids/axiom/actions/runs/41219368248/job/124590513712)
and [delivery metadata](https://github.com/rgomids/axiom/actions/runs/41219368386/job/124590514347)
are distinct from final-SHA CI, which the receipt reads separately.

Private preservation was reverified before handoff: manifest digest, all 37
included source copies and their modes, and the separate quarantined binary's
byte hash all still matched. Snapshot, quarantine and raw logs remain private;
only sanitized inventories and measured results enter this repository.

Second full `go test ./... -count=1 -timeout=30m`: **PASS**, exit 0;
`cmd/lingo` 474.838s, `internal/local` 238.561s,
`internal/workflow` 109.338s, all remaining packages passed/no test files.
No additional production edits followed the CI-driven unused-code cleanup.
No vendor CLI battery ran; fixtures were stubs and isolated temporary state.

## Review disposition

Direct engineering/security review used `axiom-review` and `axiom-document`.
No delegation was needed for these coupled recovery edits. Required unresolved
findings remain **major / blocking for S1 review readiness**, confidence high:

| Category / location | Requirement / impact | Recommendation |
| --- | --- | --- |
| specification, `cmd/lingo/execution_run.go` resolver | Plan TD-02 local defaults are not wired publicly; configured checks/before-gates refuse and per-agent bounds/dependencies need full mapping | Finish mapping with explicit provenance and exact checks; retain refusal until supported |
| architecture, `cmd/lingo/stage_service.go`, `stage_publish.go`, `delivery.go`, old workflow lifecycle | ADR-0037 forbids active legacy ownership after cutover; readers/stores/lifecycle paths remain | Complete scoped A–H/K consumer retirement; no cutover or S1-complete claim |
| verification, `internal/project/workflow.go`, `internal/projectapp/active_workflow.go` | Architecture validators fail on preexisting imports | Reconcile within remaining S1 work or retain an explicit unresolved finding; do not weaken allowlists |

The new snapshot, output/redaction, retry disposition and cutover file-boundary
corrections passed their scoped negative tests. Those results do not waive the
blocking completeness findings above or certify real installed-state cutover.
Next gate remains `resume_implement`; no new human transition is requested.
