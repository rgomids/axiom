# S1-02 implementation checkpoint — AXM-42 (PARTIAL)

Attempt `S1-02`, `implement`, [PR #366](https://github.com/rgomids/axiom/pull/366),
branch `spec/axm-42-minimum-viable-workflow`. Initial local and fetched remote
HEAD: `173ce7dd4143f5e7f44a7e3ea4ba963518fa8dd9`; verified PR base:
`main` at `440e44a68638e08e901fd3f87f5b15cd395a6365`.
Authority: the S1-02 dispatch, including the human S1 approval reference supplied
there. Linear was not accessed or published; Maestro owns publication/read-back.
This checkpoint does **not** establish S1 completion, review readiness or Product
Accepted. See [S1-01](implementation-s1-01.md) for the preceding implementation.

## Delivered code

`internal/stageexecution/runtime_launcher.go` implements the existing `Launcher`
interface for Codex and Claude using the existing process-invocation primitives.
It consumes the persisted attempt's executable/model/effort/instructions, applies
`CheckFrozenSelectors`, uses an exact filtered child environment, and carries the
attempt/generation/task and output declarations as bounded JSON on stdin. Build
never invokes a vendor command or authentication/status probe. Explicit effort
uses the existing adapter encoding; `runtime-default` sends no effort selector.

Each attempt gets a new private inbox outside the repository. Required outputs
are bounded regular files with declared names; collection hashes their actual
bytes. Missing files remain missing and the aggregate rejects successful exit
without required outputs. Symlinks, special files, oversized files, reused inboxes,
unsafe local paths and changed instruction digests refuse. POSIX opens use
no-follow/nonblocking flags and verify file identity. Other platforms fail closed
until equivalent file-handle safety exists. This is local process supervision,
not a claim that vendor sandboxes prevent every external effect.

The engine now checks persisted cancellation again after launcher preparation,
before spawn. Launcher errors are replaced with a fixed repair diagnostic rather
than storing arbitrary error text. Nonzero vendor exits produce a fixed auth/option
repair diagnostic; raw captured stdout/stderr never becomes an Execution diagnostic.

An ordinary explicit retry reuses the previous invocation and provenance despite
changed model/effort/instruction defaults, while still resolving current authority.
A changed Runtime, executable or repository refuses rather than silently retargeting.
The domain also refuses replacing the invocation through ordinary `PrepareAttempt`
retry. An explicit newly authorized selection path remains part of pending public
CLI integration; this checkpoint does not claim it is implemented.

## Per-task state

| Task | State at this checkpoint | Evidence / remaining obligation |
| --- | --- | --- |
| S1.1 | Previously delivered; preserved | ADR-0037; fresh structural ADR check below, not a new approval |
| S1.2 | Previously delivered; bounded retry correction | `execution_transitions.go`, `TestOrdinaryRetryCannotReplaceFrozenInvocation`; no new codec/store |
| S1.3 | Previously delivered resolver; consumed by launcher | `CheckFrozenSelectors` called before returning an invocation; no auth preflight |
| S1.4 | Bounded corrections delivered | cancel-during-Build, launcher-error redaction, retry snapshot/retarget tests |
| S1.5 | **PARTIAL** | concrete internal launcher and two-stage file-output fixture; **public workflow run/context/default wiring and canonical public status/cancel/retry/reconcile remain absent** |
| S1.6 | Previously delivered; preserved | no S2 transport or collaboration claim |
| S1.7 | **NOT_STARTED** | no legacy producer/consumer deletion; no old operational-path absence claim |
| S1.8 | **PARTIAL from S1-01** | no real cutover; command/installed skill/architecture reconciliation and remaining cutover negatives still pending |

The new launcher is an internal implementation seam, not an installed public
command. Existing public commands still reach legacy owners. No command/help or
installed skill documentation is changed to imply otherwise. The A–H/K owners,
retired commands and their operational consumers must be removed after public
replacement/invariant coverage exists; none is certified indispensable by this
checkpoint. Reused process primitives remain in `executiongraph` pending that work.

## Validation

Implementation source SHA: `1ebdffb5c22924d9283520b939757f49db76c923`.
The receipt-only successor commit adds this Evidence; its exact final HEAD is
reported by the S1-02 console callback (a commit cannot embed its own hash).

| Command | Actual result |
| --- | --- |
| `go build ./...` | PASS, repeated after final production changes |
| `go vet ./internal/... ./cmd/...` | PASS, repeated after final production changes |
| `go test ./internal/stageexecution ./internal/runtimeadapter ./internal/workflow ./internal/local ./internal/runtimeapplication -count=1` | PASS (initial implementation checkpoint; later affected-package checks below) |
| `go test ./internal/stageexecution ./internal/workflow -count=1` | PASS (75.325s / 447.565s); final domain production code |
| `go test ./internal/stageexecution -count=1 -timeout=30m` | PASS (61.120s), final launcher code including isolated selector-root lookup |
| `go test ./internal/workflow -run '^TestOrdinaryRetryCannotReplaceFrozenInvocation$' -count=1` | PASS, includes the added byte-identity rejection fixture |
| `go test ./internal/stageexecution -run 'TestRuntimeLauncher\|TestEngineRuntimeLauncher\|TestEngineCancelDuringBuild\|TestEngineRedacts\|TestEngineRetry' -count=1` | PASS, including new unsafe-path and Runtime-retarget negatives |
| `go test ./... -count=1 -timeout=30m` | PASS, all packages; `cmd/lingo` 1,227.801s, `internal/local` 271.487s, `internal/workflow` 196.010s |
| `python3 scripts/check-adr-governance.py` | PASS, 38 ADRs |
| `python3 scripts/check-automation-registry.py` | PASS, 99 surfaces |
| `bash scripts/check-claude-bootstrap.sh --skill-adapters .` | PASS |
| `go run scripts/check-architecture.go domain` | FAIL, existing non-allowlisted `encoding/hex` import |
| `go run scripts/check-architecture.go application` | FAIL, existing non-allowlisted `strconv` import |
| `git diff --check` | PASS |
| `bash scripts/check-sensitive-files.sh .` | PASS |
| `bash scripts/check-sensitive-files.sh --staged .` | PASS for implementation commit; repeated for receipt before commit |

The repository-wide run started during implementation; final affected production
packages were subsequently rerun as listed, with the added domain fixture also run
separately. This is composite local evidence, not a claim that a pristine final
commit ran every package simultaneously.

Repository command/help/installed-skill Go contract suites are included in the
full repository run; no separate skill test battery or real vendor CLI is claimed.
Initial-head CI was freshly read as SUCCESS for
[PR Checks](https://github.com/rgomids/axiom/actions/runs/41174385838/job/124458798700)
and [delivery-metadata](https://github.com/rgomids/axiom/actions/runs/41174385837/job/124458798643)
at `173ce7d`; these are **not** final-head CI proof. Final-head CI status is
reported separately in the console callback after push.
All new process fixtures are local stubs; no real Codex or Claude CLI was invoked.

### Preliminary failures corrected / command diagnostics

- Initial launcher conflict fixture used a model setting that the reused effort
  reader does not classify as an effort override; corrected to an actual competing
  effort selector. Explicit model argv retains precedence over that vendor default.
- Initial two-stage launcher fixture granted `workspace_write` beyond the second
  agent's Workflow bound. The engine correctly refused; the fixture now declares
  that effect on both agents.
- `bash scripts/check-claude-bootstrap.sh` without repository adapter mode failed
  (`unapproved Claude artifact: .claude`); rerun with `--skill-adapters .` passed.
- `go run scripts/check-architecture.go` without a layer failed usage. Correct
  layer invocations fail on `encoding/hex` in `internal/project/workflow.go` and
  `strconv` in `internal/projectapp/active_workflow.go`. Both imports already exist
  at initial HEAD and neither package/tool is modified here. These failures remain
  unresolved; they are not reported as passing architecture checks.

## Scoped engineering/security review

Reviewed against ADR-0037 invocation ownership, ADR-0028 process-boundary reuse,
Spec 015 stop/retry/output obligations and the security policy. The new path has
no shell interpolation, predictive auth command, automatic vendor fallback,
Provider write or legacy-store writer. Arbitrary launcher errors and child output
do not become durable diagnostics. Tests cover output/path/selector refusal and
byte-preserving retry rejection. No blocking defect was identified in the scoped
new code after correction; public integration, structural retirement, repository
validator failures and S1-wide verification remain open. Runtime sandbox behavior
and hostile concurrent filesystem mutation beyond the file-open controls are not
certified by these fixtures. No real installed-state operation was performed.

## Deletion measurement

Measured union: exact candidate paths in `deletion-baseline.json`, plus every added
Go file under `internal/` and `cmd/` since its `source_sha` (including untracked new
files during measurement). Physical lines include comments/blanks. Exported
production declarations use the Plan regexes; test functions are excluded from
production API counts. This includes S1-01 replacements, so relocation/additions
cannot masquerade as deletion.

| Metric | Baseline | Checkpoint | Delta |
| --- | ---: | ---: | ---: |
| Production files | 88 | 101 | +13 |
| Production physical lines | 21,872 | 24,578 | +2,706 |
| Production exported types | 287 | 336 | +49 |
| Production exported funcs/methods | 351 | 398 | +47 |
| Test files | 84 | 92 | +8 |
| Test physical lines | 18,320 | 20,126 | +1,806 |

Deleted production files/lines/types in S1-02: **0**. Added production files in
S1-02: launcher plus two platform flag files; additions to existing engine/domain
are included above. Legacy operational owners, writers, formats and commands
remain. The end-of-S3 net reduction is not an S1 gate; S1.7 retirement itself is
still required and has not occurred.

## technical_decisions[]

```json
[
  {
    "question": "How should concrete launchers consume the canonical frozen attempt?",
    "chosen_option": "One RuntimeLauncher implementing the existing Launcher seam for Codex and Claude, reusing BuildStageInvocation and CheckFrozenSelectors",
    "rationale": "Keeps one process boundary and one selector reader, without Profile-v4 admission or auth prediction. Separate vendor launcher types would duplicate the same preparation and collection controls.",
    "evidence_refs": ["internal/stageexecution/runtime_launcher.go", "TestRuntimeLauncherBuildsFrozenSelectorsWithoutVendorCalls", "TestEngineRuntimeLauncherProgressesTwoStagesWithFileOutputs"],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  },
  {
    "question": "How should an ordinary retry handle changed local defaults?",
    "chosen_option": "Reuse the prior invocation/provenance; recheck current authority and refuse Runtime/executable/repository retargeting",
    "rationale": "Retry intent alone cannot authorize a new selection. A domain guard prevents bypass by another caller. Newly authorized selection remains a separate pending integration obligation.",
    "evidence_refs": ["TestEngineRetryPreservesFrozenSelectionAfterLocalDefaultsChange", "TestEngineRetryRefusesRuntimeRetargeting", "TestOrdinaryRetryCannotReplaceFrozenInvocation"],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  },
  {
    "question": "How should declared local outputs be collected without treating exit zero as success?",
    "chosen_option": "Read only declared bounded regular files from a new private attempt inbox; hash bytes and leave missing outputs absent",
    "rationale": "Uses the existing artifact-reference aggregate and adapter inbox. No new result store, stdout success parser or substitute certification. Fail closed without safe no-follow file primitives.",
    "evidence_refs": ["TestRuntimeLauncherCollectsOnlyDeclaredBoundedRegularFiles", "TestRuntimeLauncherRefusesUnsafeMachineLocalInputs"],
    "scope_impact": "none",
    "decision_maker": "executor-ai"
  }
]
```

## deferred_findings[]

- Existing project/projectapp architecture import-allowlist failures above. Evidence:
  correct layer invocations and the same imports at `173ce7d`. Reconcile during
  scoped repository validation; no allowlist weakening performed here.
- Historical Model Advisor and builtin SDD drift from S1-01 were not investigated.
- PR title/body still describe documentation-only work and say implementation has
  not started (read back at initial HEAD). This contradicts S1-01/S1-02 code.
  PR metadata mutation is outside the dispatch's explicit commit/push exception to
  remote mutations; leave for the authorized publisher's pre-merge reconciliation.

## Remaining work / next gate

1. Complete S1.5 public `workflow run`, linked Project/Work Item/Repository context,
   Workflow/default and per-agent local defaults, canonical status/cancel/retry/
   reconcile, explicit new-selection authorization and public-entry fixtures.
2. S1.7: port surviving invariants, then remove A–H/K producers AND consumers,
   stores/codecs/supervisors/retired-only tests/commands; verify old writer and
   operational import/call absence; remeasure the union against baseline.
3. S1.8: finish isolated unreadable/unsafe/unknown cutover negatives and byte-identity
   evidence; reconcile architecture, command metadata/help, installed docs/skills,
   stop-or-block diagnostics and rollback boundaries for the actual public path.
4. Repeat affected checks after those changes and reconcile existing validator
   failures; engineering review remains necessary before declaring S1 complete.

Live vendor E2E, Provider/remote effects, real installed/private state inspection,
backup, process stop or cutover, credential use, merge/release and Product Accepted:
**NOT_RUN**, outside this authorization. Race: **NOT_RUN**, no C compiler installed
(`CGO_ENABLED=0`, configured `CC=gcc` but no `gcc` in PATH). Gitleaks: **NOT_RUN**,
not installed. No tools/plugins were installed and no callback was invoked.

Next gate: **resume_implement**. Checkpoint: **AWAITING_EXECUTION_REPORT** for Maestro
publication and API read-back; `linear_published:false`. No human transition approval
is requested for the remaining already-authorized S1 work.
