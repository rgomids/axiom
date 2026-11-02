# S2.3 — Isolated agent worktrees and principal-controlled integration

Attempt: `IMPLEMENT-S2-03-CLAUDE` (Claude Code, `claude-opus-5-5`). Bounded
S2.3 checkpoint on the existing AXM-42 / PR #366 branch. It does not claim
S2.4, S3, S2 integrated closure, human acceptance, merge, release or cutover.

## Authority and preflight

[Task S2.3](tasks.md) depends on S2.1 only
([S2.1 receipt](https://linear.app/rgomids/issue/AXM-42#comment-e429cb40)).
S2.2 / S22-CR-01 has an independent Codex analytical
[APPROVE](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5742024731)
([receipt](https://linear.app/rgomids/issue/AXM-42#comment-5d740e7f)). The
separate **ChatGPT independent CR of S2.2 is PENDING**: not completed, not
waived. It is not an S2.3 implementation gate, but it remains an open review
dependency for S2 integrated reconciliation and pre-merge. No new S2.2 finding
was communicated during this attempt.

Preflight on `ubuntu-arm-01`, worktree `~/axiom-wt/axm-42`: branch
`spec/axm-42-minimum-viable-workflow`; local HEAD, fetched origin and PR #366
head all equal the pinned `95cfa3d2dcc5458af7cd9653b52a8c49e8bb59ff`; PR Draft;
clean index/worktree including untracked; no running Git process. The other
registered worktree (`~/axiom`) is a separate branch and was not touched.

## Contract mapping

| Source | Requirement | Realization |
| --- | --- | --- |
| Spec 015 "Parallel work and integration" | Recommend an isolated worktree and branch per agent; the principal owns the integration branch, serializes merges, resolves nothing silently, runs post-merge checks and records final status before integrated; no Git subsystem | `gitworkspace.Integrator.PrepareAgentWorktree` (opt-in, new path + new branch only); `stageexecution.Engine.Integrate` (principal-only, runner lease) |
| Plan TD-03 | Record repository identity, base SHA and agent result SHA; expected integration HEAD; dirty/changed base or conflicts stop; failed checks remain not_integrated; no destructive reset/cleanup | `workflow.Integration` nested in Execution format 4: repository key + bound identity check, source attempt/stage/agent, branch, worktree name, base, target ref, expected and observed target head, candidate, intermediate merge, final SHA, checks, status, runner generation |
| Plan TD-01 / ADR-0037 §1–3 | One aggregate, one store, one fenced engine; persist intent before effect; crash without correlated evidence is unknown, never re-run | Intent and outcome are two fenced revision-CAS commits; `Engine.Claim` marks an older runner's prepared integration `unknown` |
| ADR-0037 §6 | Messages never grant authority | Integration has no message path; test proves a child `result` message claiming integration changes nothing |
| Tasks S2 note | Do not guarantee conflict-free worktrees; preserve user branches; actual workspace authority | Conflicts are recorded; no ref/branch/worktree is ever deleted or reset; the composition supplies the exact authorized repository and worktree root |

No new public CLI flag, command, store, schema family, owner, Graph or Git
service. `Integrations` is an optional (`omitempty`) member of the existing
format-4 aggregate; Executions without integrations keep their exact bytes.

## Affected producers and consumers

- `internal/workflow/execution_integration.go` (new): record, statuses,
  `PrepareIntegration`, `FinishIntegration`, `MarkIntegrationUnknown`,
  `ResolveIntegration` (human abandonment), `AcceptedTargetHead`, validators.
- `internal/workflow/execution.go`: `Execution.Integrations`; `Validate` calls
  `validateIntegrations`. `execution_transitions.go`: an unresolved unknown
  integration derives `recovery_required` (blocks `Ready` and new attempts).
- `internal/gitworkspace/integration.go` (new): `Integrator`, reusing the
  existing `Manager` runner (`core.hooksPath=/dev/null`, no system/global
  config, no credential helper, executable-configuration veto), rooted
  no-symlink path confinement and no-cleanup property. `git.go`:
  `merge.<driver>.driver` added to the executable-configuration veto because
  `merge-tree` consults merge drivers (applies to the legacy Manager too:
  strictly more refusals).
- `internal/stageexecution/integration.go` (new): `Engine.Integrate`,
  `IntegrationGit` port, post-merge checks via the existing `ObservedRunner`
  (ADR-0028 process group) and the existing command gate. `engine.go`: `Claim`
  records orphaned prepared integrations unknown.
- No production caller is added: the CLI, `RuntimeLauncher` (single
  `Workspace`) and S2.2 ingress are unchanged. Consumers of the new API are the
  tests below. Composition into a public surface is deferred (DF-S23-01).

## Behaviour

`Engine.Integrate` claims the Execution's runner lease (so it is serialized
with `Run` and every other integration of that Execution; a child attempt
holds no lease) and then:

1. observes the candidate: registered worktree of the *same* common directory,
   on exactly the named branch, no tracked or untracked change, tip descends
   from the recorded base and differs from it; observes the target: exists, not
   checked out in any worktree, equals the expected head. Any refusal here
   records nothing and has no Git effect (`ErrIntegrationRefused`);
2. commits the intent under the fence. The domain refuses: stale fence, cancel,
   outstanding unknown (attempt or integration), unbound or changed repository
   identity, a source attempt that is not `succeeded`, lacks
   `workspace_write`+`validation_command` or belongs to another repository, a
   source stage with no check (`ErrIntegrationChecksUnavailable`) or an
   unapproved human gate, another prepared integration (`ErrIntegrationBusy`),
   an expected head different from the last recorded `FinalSHA` for that
   target (`ErrIntegrationStale`), a re-integration of an integrated
   candidate, and reuse of a branch/worktree/attempt association;
3. computes the merge with `git merge-tree --write-tree` + `commit-tree`
   (no ref, index or working tree touched). Conflict → `not_integrated`
   `merge_conflict`;
4. checks the intermediate merge out, detached, in a new principal worktree
   `integration-<id>` and re-runs every source-stage command there with an
   exact minimal environment and the command's own timeout. Failure, timeout,
   spawn failure or unobservable outcome → `not_integrated`
   `post_merge_check_<failed|terminated>` (MergeSHA retained, FinalSHA empty);
5. re-verifies the check worktree unchanged (`check_mutated_worktree`), no
   interruption and no persisted cancel (`cancelled_before_effect`);
6. compare-and-swaps the target with `git update-ref <ref> <merge> <expected>`
   and re-observes it. Observed merge → `integrated` (FinalSHA = MergeSHA =
   observed). Refused CAS with the target at another commit →
   `not_integrated` `target_moved_concurrently` / `target_advance_refused`.
   Unobservable target, or an error while the target shows our merge →
   `unknown` (`advance_outcome_unknown` / `advance_outcome_ambiguous`).

The intermediate merge (MergeSHA, retained worktree) is distinct from accepted
finalization (target CAS observed + FinalSHA recorded). A `not_integrated`
record means the target was not advanced by this integration; nothing is
rolled back and no rollback is claimed. Nothing is retried automatically:
`unknown` blocks further integrations and dispatch until a human abandonment,
which keeps the uncertain record unchanged.

### technical_decisions[]

| ID | Choice and reason | Alternatives / trade-off | Rollback |
| --- | --- | --- | --- |
| TD-S23-01 | Nested optional `Integrations` in Execution format 4 (Plan TD-01/TD-03 "record … before integrated") | Encode into `EffectFact`/`ArtifactRef` (loses base/expected/check facts); separate record/store (forbidden second owner) | Pre-release aggregate; revert the commit, no persisted data exists |
| TD-S23-02 | `merge-tree --write-tree` + `commit-tree`, then ref CAS | Merge in a checked-out integration worktree (mutates a tree that a failed check must then reset); cherry-pick (rewrites agent SHAs, loses candidate identity) | Git ≥ 2.38 required; older Git fails closed (`merge_failed`) |
| TD-S23-03 | `update-ref` old-value CAS serializes the shared destination across processes; an in-process mutex only orders observation | A lock file or second scheduler (new subsystem); a mutex alone (process-local) | None needed; CAS is Git's own ref transaction |
| TD-S23-04 | Refuse a target checked out in any worktree | Update it anyway (desynchronizes and could overwrite user work); reset that tree (destructive) | Operator detaches/removes their own checkout first |
| TD-S23-05 | Post-merge checks = the source stage's Workflow `Commands`, at least one required | New check field/authority (unapproved field); optional checks (integrates without check Evidence) | A stage without commands cannot be integrated |
| TD-S23-06 | Integrate claims the runner lease (like `Reconcile`) | Pass a fence into the running loop (couples integration to dispatch); lease-free CAS (lets two principals interleave) | Integration waits while a `Run` holds the lease |
| TD-S23-07 | Never remove agent or check worktrees | Automatic cleanup (risks user/forensic state) | Accumulation; disposal stays an explicit operator action |

All are reversible TECHNICAL_LOCAL choices inside Plan TD-01/TD-03 and
ADR-0037; no material product/architecture/design decision and no dependency
was introduced.

## Deterministic proof

Every Git repository is created with `t.TempDir()` (a fresh repo with `a.txt`,
`b.txt`, `shared.txt`, branch `axiom/integration` at base, an unrelated
`user/keep` branch with a dirty user worktree) and removed by the test
framework. No remote exists; each scenario asserts `git remote` is empty, no
`refs/remotes/*` appeared, no pre-existing ref was removed or moved (except the
target where the scenario integrates), no worktree was removed and the dirty
user worktree is byte-identical. Agent attempts are in-process stubs that edit
and commit inside their own worktree; only post-merge checks are real
processes. No user working repository is a fixture and no vendor CLI runs.
`assertNoChildProcesses` (from S2.2) checks `/proc/self/task/*/children` after
each fixture: no test-owned process survives. SHAs are random per run (commit
timestamps), so lineage is asserted structurally on every run.

`internal/stageexecution/integration_test.go`:

| Test | Scenario → asserted outcome |
| --- | --- |
| `TestIntegrateParallelWorktreesSequentiallyWithCheckEvidence` | Barrier requires both agents active (max active = 2) while editing independent files on distinct branches from the same base; principal integrates A then B; records carry candidate/base/expected/final, passed check; final = merge(finalA, candB), finalA = merge(base, candA); `main`, agent branches stable; both check worktrees retained; replaying A refused with target unchanged |
| `TestIntegrateSameFileConflictIsExplicit` | Same-file edits: B `not_integrated` `merge_conflict`, no MergeSHA/FinalSHA, observed head = A's final, no check worktree |
| `TestIntegrateStaleBaseAndDirectTargetDriftRefuse` | Expected head = base after A integrated → refused stale, nothing recorded; a child-style direct `update-ref` of the target is detected (expected ≠ observed) and the drifted head is not adoptable against recorded lineage; recorded state unchanged |
| `TestIntegrateDirtyCandidateAndCheckedOutDestinationRefuse` | Tracked and untracked dirty candidates refused (`ErrForeignMutation`), dirty files preserved; target checked out with uncommitted work refused (`ErrIntegrationTargetCheckedOut`), that work preserved; zero records |
| `TestIntegrateRepositoryBindingAndUnauthorizedSources` | Changed repository identity refused; failed attempt with a commit refused; worktree of another repository under the authorized root refused (common directory); refusals record nothing and do not prevent the authorized integration |
| `TestIntegrateInvalidCandidateAndEscapedWorktree` | Branch without a commit beyond base (`ErrNoResult`); invalid/abbreviated/unrelated base, `../` and absolute worktree names, symlinked worktree, branch mismatch, traversing/lock refs, missing target, missing attempt, unbound repository key, empty expected head: all refused, zero records, target unchanged |
| `TestIntegrateFailedPostMergeCheckStaysNotIntegrated` | Each change passes alone; together the check fails: `not_integrated` `post_merge_check_failed`, exit 1 recorded, MergeSHA is a commit retained in its check worktree and not reachable from the target, no "rollback" claim, not `recovery_required` |
| `TestIntegrateConcurrentRaceOnSharedDestination` | Two Executions with two `Integrator` instances held at `Advance` until both arrive: exactly one `integrated`, the other `not_integrated` `target_moved_concurrently` observing the winner's final; loser's merge not reachable from the target |
| `TestIntegrateUnknownGitOutcomeIsNeverRetried` | Advance error with unobservable target → `unknown` `advance_outcome_unknown`, Execution `recovery_required`, next integration refused `ErrUnknownOutstanding`, exactly one advance; a CAS that succeeded but reported an error → `unknown` `advance_outcome_ambiguous`, never `integrated` |
| `TestIntegrateLostOutcomeBecomesUnknownForNextRunner` | Outcome commit lost after the Git effect: record stays `prepared`; the next runner (`Reconcile`) records `unknown` `runner_lost_before_integration_outcome` and does not touch the target |
| `TestIntegrateHumanGateAndCancelProhibitEffect` | Unapproved human gate refuses intent; cancel persisted after checks → `not_integrated` `cancelled_before_effect`, zero advances, target unchanged |
| `TestIntegrateCheckTimeoutStopsOwnedProcess` | `sleep 30` check with 1 s timeout → `post_merge_check_terminated`; the observed check process identity is gone afterwards |

`internal/workflow/execution_integration_test.go`: codec round trip and
canonical bytes, forged `integrated` status refused on decode, empty
integrations not encoded; outcome invariants (integrated needs all checks
passed and target observed at FinalSHA = MergeSHA); serialization (one
prepared); lineage stale; association mismatch; 18 intent refusals each leaving
the encoded Execution byte-identical (stale/zero fence = child without fence,
unknown/running attempt, wrong repository, changed identity, missing,
abbreviated, uppercase or base-equal candidate, source = target, tag/
traversing/lock refs, escaping/hidden/nested worktree, invalid ID); unapproved
gate and agent gate approval; cancel; lost runner → unknown → blocks Ready and
intent; agent cannot resolve, human abandonment keeps the uncertain record and
permits an explicit new integration; no-check stage refused; a child `result`
message claiming integration creates no state.

`internal/gitworkspace/integration_test.go`: a control repository proves a
`merge.<driver>.driver` is live under `merge-tree`; the Integrator refuses it
(`ErrExecutableGitConfiguration`) both at binding and when configured after
binding, without the sentinel being created. Existing user branch and
existing worktree path are refused (never reused); escaping/hidden/nested
names refused; root inside repository, repository inside root and a
non-top-level path refused.

### Red/green record

- First run of the engine matrix: 10/12 failed with
  `post_merge_check_terminated`: the fixture used `/bin/sh`, which is a symlink
  on this host and is correctly refused by `cleanRuntimePath`. The fixture now
  resolves the symlink-free shell path; production code unchanged. One further
  failure was the fixture's own extra commit taken after the snapshot (moved
  before it). Gitworkspace control test initially did not prove the driver live
  (fixture detached at a commit without `.gitattributes`); corrected. Domain
  tests passed on first run.
- Mutation checks (each applied alone, observed red, reverted, green again):
  M1 `update-ref` without old value → race test `won=2`; M2 ignore failed
  checks → failed-check and timeout tests red (the domain validator also
  refuses the forged `integrated` record); M3 disable lineage check → drift test
  red and domain lifecycle test red; M4 disable checked-out target refusal →
  destination test red.
- `go test ./internal/workflow ./internal/gitworkspace ./internal/stageexecution -run 'TestIntegrat' -count=3 -timeout=10m`: exit 0 (stageexecution 34.8 s).

## Security and failure limits

- Same-UID agents share the repository's refs and object store: a child can
  move a ref directly. This is detected (drift refusal; never recorded or
  adopted), not prevented; OS confinement remains an ADR-0028 limit
  (DF-S23-02, analogous to DF-S22-04).
- The branch ↔ attempt association is asserted by the principal; the domain
  only prevents reusing a recorded association for another attempt.
- Post-merge check process identity is observed in-run but not persisted; a
  runner crash during a check leaves `prepared` → `unknown` without a check PID
  to re-observe later (DF-S23-03).
- `git` subprocesses run under `exec.CommandContext` without process-group
  containment (existing `Manager` behaviour, DF-S23-06).
- No transactional rollback across Git processes exists or is claimed;
  `update-ref` is the only ref write and is a single CAS.
- No push, fetch, remote, credential, PR, vendor CLI, installation or cutover.

## Validation

Pre-commit snapshot of this worktree delta from `95cfa3d` on 2026-11-02, Go
1.27.3 (`/usr/local/go`, matched GOROOT, `GOTOOLCHAIN=auto`, module `go
1.26.0`). The final receipt binds the committed SHA and exact-SHA CI.

| Command | Result |
| --- | --- |
| `go build ./...` | PASS, exit 0 |
| `go vet ./...` | PASS, exit 0 |
| `gofmt -l cmd internal scripts` | PASS, no output (after formatting one test helper) |
| `go run ./scripts/check-architecture.go domain` / `application` | PASS, exit 0; 12 / 46 files |
| `go test ./... -count=1 -timeout=30m` | PASS, exit 0; 32 packages `ok`, none failed (`cmd/lingo` 149.6 s, `workflow` 76.5 s, `stageexecution` 44.0 s, `gitworkspace` 10.5 s); ambient PATH. An earlier identical run was stopped by the executor after two packages to attempt a vendor-free PATH, not a failure |
| `PYTHONDONTWRITEBYTECODE=1 bash scripts/validate-repository.sh .` | FAIL, exit 1: only `test_model_advisor_eval.py` `test_unavailable_runtime_exits_nonzero` (`0 != 1`), because real `claude`/`codex` are on the ambient PATH (`~/.local/bin`). Known historical Model Advisor synthetic failure / validator PATH risk; S2.1 passed with a vendor-free PATH. Earlier stages printed PASS (WARN empty directories; runtime behavioral scenarios SKIP/UNVERIFIED); stages after the failure did not run |
| `scripts/check-sensitive-files.sh .` | PASS, exit 0 |
| `git diff --check` | PASS, exit 0 |
| Vendor-free PATH (`PATH=/usr/local/go/bin:/usr/bin:/bin …`) | NOT_RUN: this session's permission policy refused the PATH override; the `cmd/lingo` Runtime fixtures pin `PATH` to their own temp stub directory |
| `./scripts/check-go-quality.sh all` (Staticcheck), `check-adr-governance.py`, `check-automation-registry.py`, `check-claude-bootstrap.sh` direct | NOT_RUN: refused by this session's permission policy (needs `AXM_EXTRA_TOOLS`); the bootstrap behaviour was exercised inside `validate-repository.sh` (PASS) |
| Remote CI on `6a3757d` | FAIL ([run 41513612190](https://github.com/rgomids/axiom/actions/runs/41513612190)): Staticcheck SA4006 unused `x` at `internal/stageexecution/integration_test.go:519`, the only finding; later CI steps skipped. Corrected in the next commit (test-only), whose exact-SHA CI is reported in the PR comment/receipt. Delivery metadata on `6a3757d`: PASS |
| `go test -race` | NOT_RUN: no `gcc` / cgo on host (as S2.1) |
| `gitleaks` | NOT_RUN: not installed |

## deferred_findings[] (DEFERRED_NOT_IMPLEMENTED)

- DF-S23-01: no public surface composes `Integrate`/`PrepareAgentWorktree`;
  `RuntimeLauncher` still runs agents in one `Workspace`. Wiring requires a
  CLI/launcher contract (no new flag was permitted here); candidate for S2.4/S3.
- DF-S23-02: same-UID ref writes by children (detected, not prevented).
- DF-S23-03: check process identity not persisted across runner loss.
- DF-S23-04: agent and check worktrees accumulate; disposal is an explicit
  operator action.
- DF-S23-05: `merge-tree --write-tree` needs Git ≥ 2.38 (host 2.43.0); older
  Git fails closed, untested here.
- DF-S23-06: Git subprocesses lack process-group containment (pre-existing).
- DF-S23-07: `ResolveIntegration` and integration status views have no public
  command yet (S2.4/S3.1).
- Inherited, not re-proven here: DF-S22-01..04 (S2.2) and the S2.1/S1 deferred
  set recorded in their Evidence.

## Remaining work and next gate

S2.3 technical scope is implemented as library operations with isolated
evidence. Next gate: **INDEPENDENT_REVIEW_S2_3** and
**RECONCILE_PENDING_S2_2_CR** (independent ChatGPT review of S2.2) before S2
integrated closure; S2.4 is not dispatched by this attempt. Human pre-merge
review, merge, release, Product Accepted and AXM-12 remain separate human
gates. No background continuation exists; checkpoint
`AWAITING_EXECUTION_REPORT`.
