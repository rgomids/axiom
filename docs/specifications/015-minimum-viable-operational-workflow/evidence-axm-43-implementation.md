# AXM-43 — Same-repository integration implementation Evidence

2026-11-11. Base: `8df55e24ceb7e61bfebb1a70a9f64436b2c8c0d1`
(approved amendment, PR #390). Scope: option A, Tasks 1–5 of the
[amendment](amendment-axm-43-same-repository.md). This is a bounded technical
implementation with the decided refusal contract below, not full
AXM-43 completion, P-AC3 Product acceptance, AXM-1/#278 acceptance or cutover.

## Decision (Rafael, 11/11/2026 16:07 BRT)

Literal: “Mantém a recusa e move o gate de revisão para um Stage separado depois
da integração” ([decision record](https://github.com/rgomids/axiom/pull/396#issuecomment-6804859927)).
Option 1 below is chosen; options 2 and 3 are not chosen. This is a
clarification within approved option A, not P-AC3, AXM-12, AXM-1 or #278
acceptance. The refusal now returns a dedicated diagnostic (category
`integration_refused`) whose next step directs the operator to put the
review/acceptance gate on a separate later Stage that runs after integration.

Context recorded while the decision was pending:

An after-Stage gate and a same-Stage dependent consuming an enrolled writer
form a cycle: `PrepareIntegration` requires the human gate; the dependent needs
integration; the after gate requires all Stage results. Exposing that gate
after the first writer succeeds would let `human_acceptance` approve results
which do not exist yet. The amendment requires preserving gates and does not
authorize redefining their timing or bypassing integration admission.

The implementation therefore **refuses this combination before any attempt,
Git ref/worktree or Runtime spawn**. Existing unenrolled gate semantics remain
unchanged. A normal after gate can still approve all completed independent
results before integration; a durable known gate-wait marker distinguishes that
intentional stop from a lost principal. No approval is synthesized.

Options presented to Rafael:

1. **Chosen.** Preserve the refusal; configure the review/acceptance gate in a
   separate downstream Stage after integration (preserves existing timing).
2. Not chosen. Authorize a distinct pre-integration approval meaning, with explicit semantics
   for still-unexecuted dependents; this changes the gate contract.
3. Not chosen. Authorize integration before the after gate while retaining that gate for
   final Stage results; this changes accepted integration admission.

Gate timing and persisted approval meaning become harder to change once real
Executions use them. Before the decision, work stopped at this material decision
as instructed; only safe coherent implementation, validation and authorized PR
delivery continued. The PR body and PR comments record the decision. No new ADR, public command, flag or state family was
introduced or accepted.

## Implementation and acceptance mapping

Every named deterministic test below is **PASS**. Tasks 1–2 and 4 have their
scoped technical evidence; Tasks 3 and 5 are **PASS for the bounded subset**,
with the after-gate/dependent combination decided as a refusal with separate-Stage
guidance. No table entry claims full AXM-43 or Product acceptance.

| Task / contract | Implementation | Deterministic Evidence |
| --- | --- | --- |
| 1: exact enrollment, optional format-4 fields, immutable fenced binding | `execution_workspace.go`, Execution codec, local transition guards | `TestWorkspace*`, `TestExecutionWorkspace*`: configured dependent writer counted, downstream consumer binding, codec/tampering/lineage, fence/immutability/snapshot isolation, canonical inventory recognition |
| 1: old unenrolled bytes and behavior | omitted optional fields; creation explicitly opts in | `TestI369LegacyBytes`; `TestEngineUnenrolledWritersKeepOriginalRunPath`: no workspace callback/integration records; historical compatibility corpus checks in full suite |
| 2: intent before Git/spawn; one attempt workspace | `workspace.go`, hardened `gitworkspace.Integrator`, `RuntimeLauncher` Build/Inspect/Collect | public parallel/dependent tests and collision/changed-base/intent-loss faults; Git planning test verifies constructor creates no root |
| 3: integration gates dependencies and completion | domain readiness/status, lease-held integration operation, principal scheduler waits for peer callbacks | public success asserts two simultaneous writer stubs, serial merge parentage and expected HEAD, post-merge check facts, dependent reads both commits; no replay on continuation |
| 3: current admission/effects | recheck before intent, checks, gated command start and ref advance | `TestIntegrateRechecksAdmissionBeforeChecksAndAdvance`; public composition archive-after-merge fault: no check/ref advance, truthful admission diagnostic |
| 3: failure and uncertainty | retained `not_integrated`/`unknown`, orphan workspace uncertainty; no automatic reset/retry | public conflict/check failure, lost workspace/result/integration-intent/advance/outcome faults; no dependent or second effect |
| 4: actual public composition and safe projection | `executionComposition.engine`, new-start enrollment, workspace factory, status/outcome/next-action projection | `TestPublicWorkflowRunSameRepository*`, `TestPublicCompositionSameRepository*`; public parser/resolver/launcher/store/Git composition, synthetic state and fake Runtime executable |
| 4: help/skills/installed metadata | CLI command summary/authority catalog; embedded workflow/work-item skills and digest history; operational docs | CLI, codexruntime and operationaldocs package checks; repository validator |
| 5: bounded negative cases | missing checks/authority, dirty/changed base, branch/path collisions, cancel, gates | public matrix asserts no successor/unauthorized/double effect; original operator HEAD/ref/files preserved; after-gate cycle refuses before any attempt |
| 3/5: after-gate dependent contract (decided: refuse, separate review Stage) | `ErrAfterGateIntegrationDependency` refusal before any attempt/Git/spawn; dedicated next step; no approval of partial Stage results | **PASS** `TestPublicWorkflowRunSameRepositoryDependentWriterAndHumanGate/gate`: category `integration_refused`, separate-Stage next step, no attempts/integrations/gate/`axiom/` refs; **NOT_RUN** manual P-AC3 acceptance |
| P-AC3 full requirement | technical path plus bounded refusal above | **NOT_RUN** human manual Codex+Claude journey; no Product acceptance claim |

All public fixtures use temporary Git repositories with no remote, synthetic
HOME/state roots and a local shell stand-in for the vendor executable. Faults
wrap ports from the actual CLI composition; they do not replace its resolver,
RuntimeLauncher, store, admission or Git implementation. No real vendor inference,
real Axiom state, forbidden worktree or Provider mutation was used.

The public matrix exposed artifact publication contention: the existing artifact
store takes a nonblocking root lock while Execution callbacks/polls also read
that root. Isolated output subjects now reuse the existing bounded `lockRoots`
acquisition before artifact read/create; no mutation is replayed. Unenrolled
subjects retain the original nonblocking acquisition. No separate lock owner or
storage protocol was added.

## Executed checks

Results are recorded only after completion. Initial synthetic fixture failures
(schema/fixture authoring and concurrent artifact lock acquisition) were corrected;
they are not counted as passing runs.

| Command / check | Result |
| --- | --- |
| `go test ./internal/workflow -count=1` | PASS |
| `go test ./internal/local -run 'TestExecutionWorkspace\|TestExecutionStore\|TestI369' -count=1 -timeout 10m` | PASS |
| `go test ./internal/gitworkspace ./internal/stageexecution -run 'TestIntegrat\|TestObserveIntegration\|TestEngineUnenrolledWritersKeepOriginalRunPath\|TestEngineParallelReadyTasksStreamAndCommitWithoutLoss' -count=1 -timeout 5m` | PASS |
| `go test ./internal/cli ./internal/codexruntime ./internal/operationaldocs -count=1 -timeout 10m` | PASS; final focused parity checks also PASS |
| `go test ./cmd/lingo -run 'TestPublicWorkflowRunSameRepository\|TestPublicCompositionSameRepository' -count=1 -timeout 10m` | PASS (57.764s), including exact fault-reached/ref-count assertions and collision preservation |
| Final focused CLI/public matrix after restricting new recovery guidance to enrolled attempts (`-run 'TestExecution\|TestPublicWorkflowRunSameRepository\|TestPublicCompositionSameRepository'`) | PASS (72.411s) |
| `git ls-files '*.go' \| xargs gofmt -l` | PASS (empty); added Go files separately formatted |
| `go vet ./...` | PASS (empty output) |
| `go test ./... -count=1 -timeout 40m` | PASS (one full run; `cmd/lingo` 326.561s, `local` 103.409s, `stageexecution` 60.608s); final focused CLI run covers the last presentation-only scope guard |
| `env PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/validate-repository.sh .` | PASS; optional real-runtime behavioral scenarios explicitly skipped/unverified |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `./scripts/check-sensitive-files.sh --staged .` | PASS before commit |
| `git diff --check` | PASS |
| `gitleaks` | NOT_RUN: not installed |
| Real vendor inference / real state / manual Product acceptance | NOT_RUN: forbidden/outside authority |

Decision correction (dedicated refusal diagnostic and separate-Stage guidance),
re-run after the change: `go test ./internal/cli ./internal/codexruntime
./internal/operationaldocs ./internal/stageexecution` PASS;
`go test ./cmd/lingo -run 'TestExecution|TestPublicWorkflowRunSameRepository|TestPublicCompositionSameRepository'`
PASS; `gofmt -l` empty; `go vet ./...` PASS; `go test ./... -count=1 -timeout 40m`
PASS (`cmd/lingo` 234.134s); repository validator (restricted PATH) PASS;
`./scripts/check-sensitive-files.sh .` PASS; `git diff --check` PASS.

## Review, deferred findings and delivery boundary

Delegation was used for independent aggregate, runner/Git and documentation work,
with the principal retaining integration and publication authority. A fresh
Ponytail/engineering review found the gate timing conflict above and an admission
refusal incorrectly labeled cancellation. Early gate timing was removed and the
combination refused; admission refusals now have their own diagnostics, preserving
unknown process outcomes. Targeted tests cover both findings.

DF-S23-01 now has a reachable public technical path, bounded by the decided
refusal above, and is **not** full P-AC3 acceptance. DF-S23-02..07 remain deferred:
same-UID shared Git mutation, check-process uncertainty after runner loss,
retained-worktree accumulation, Git version requirement, Git subprocess
containment and absence of public integration resolution. Nothing is cleaned,
deleted, force-pushed, merged, released or accepted. PR/CI evidence is reported
on the PR, not inferred from local tests.

## Real vendor journey and fixes (AXM-12, 2026-11-12)

Host macOS 27.2.1 arm64; codex-cli 0.166.0 (gpt-6-luna, low) and Claude Code
2.1.322 (claude-haiku-5-5, low); Project `axiom-mvp-e2e-20261106`, Repository
`library` at `c05e1b9`. The released 0.24.0 failed the same-Repository journey
with real vendors (Execution `710b3b2c`, `not_integrated`):

- **Codex writer could not commit.** Under `--sandbox workspace-write` Codex
  keeps `.git` read-only even inside writable roots (`touch .git/probe` and
  `git add` in a linked worktree: `Operation not permitted`), so the
  "writers commit" contract was unreachable for Codex. The principal now
  records a writer's uncommitted changes as one `axiom-principal` commit on
  exactly its branch after it exits (`Integrator.CommitPending`: hooks and
  executable drivers refused, branch moved by compare-and-swap, idempotent).
  Only a writer that made no commit is captured; leftovers after a writer's own
  commit are still refused as foreign mutation.
- **Post-merge checks could never pass for real toolchains.** They ran with
  `PATH=/usr/bin:/bin LC_ALL=C` only, so `go test` failed
  (`GOCACHE is not defined and $HOME is not defined`). They now use the same
  allowlisted command environment as the attempt's own checks
  (`runtimeadapter.CommandEnvironment`).
- Parallel attempts of an unenrolled Execution read predecessor outputs with a
  nonblocking artifact lock; contention refused one launch as
  `builder_failed` (Execution `f65d90b1`). Execution artifact reads,
  validation and retention now always use the bounded lock wait.
- A missing `--authorize-commands` now reports `commands_unauthorized` and
  names the flag; the agent contract tells writers to leave changes
  uncommitted when a commit is refused and recipients to answer pending
  questions before exiting.

Real-vendor results with the fixed build (development `f5709fc`):

| Journey | Execution | Result |
| --- | --- | --- |
| Codex + Claude writers, same Repository, `go test -count=1 ./...` post-merge check | `e0c6ba9c` | both `integrated`, checks passed; serial merges `c05e1b9 → c80decd → a8a3be6`; Codex candidate `544f5f6` recorded by the principal; dependent reader ran on `a8a3be6`; operator HEAD unchanged |
| Real merge conflict (both writers rewrite README.md) | `8215f862` | first `integrated`, second `not_integrated` / `merge_conflict`; dependent never ran; continuation replayed nothing |
| After-gate + enrolled dependent | `fa7efdb6` (0.24.0) | `integration_refused`, no attempt/ref/worktree/spawn |
| Codex + Claude multi-Repository with typed Q→A | `5e4570d9` | `cli-q1` question and `lib-a1` answer (`replyTo`) persisted with sender attempts; answer quoted in `cli-result` |
| Reject + rework | `9493e3b0` → `bec32a4d` | human rejection retained as Evidence; rework with fresh selection/authority and `reworkOf` lineage; without them refused |
| Runner SIGINT / SIGKILL | `4b19540a` / `8d1c1b2c` | `group_stop_confirmed` cancel; lost runner → `unknown`, never respawned, human abandonment pending |

Not covered here: Linux `untrusted_ancestor_owner` (no Linux host in this run),
human gate approval and Product acceptance.
