# CORRECT-399-02 — Recoverable source capture

2026-11-12. AXM-12 / [PR #399](https://github.com/rgomids/axiom/pull/399).
Starting local/remote HEAD: `1849be2e53c28ab322ff511845a75597287e764e`;
observed main/base: `c916680106ac6c35a1e7da14d3effded8b48e079`.
The worktree was clean; branch, origin and open PR matched the dispatch.
Delivery commit and verified remote SHA are reported in the completion event.

Authority: [human approval](https://linear.app/rgomids/issue/AXM-12#comment-9209212a),
[re-review](https://github.com/rgomids/axiom/pull/399#pullrequestreview-5881518999),
[dated normative exception](amendment-axm-43-same-repository.md#2026-11-12--bounded-source-capture-exception-to-revision-2-rule-4).
The [previous decision checkpoint](correction-399-capture-decision.md) remains
historical; this evidence does not approve merge, release or Product acceptance.

## Order and ownership

```text
observe target + existing admission
  -> exact enrolled/succeeded Attempt + frozen Workspace/Authority
  -> persist Attempt.capture v1 intent (exact path + runner generation)
  -> fenced checkpoint -> git add
  -> fenced checkpoint -> write-tree
  -> fenced checkpoint -> commit-tree
  -> persist exact CandidateSHA in the same Execution
  -> fenced checkpoint + exact path/branch/base/config + candidate-tree check
  -> source update-ref(candidate, expected base)
  -> read-only capture observation -> persist observed facts
  -> observe clean committed candidate -> original PrepareIntegration
  -> merge + required checks -> target CAS -> original integration outcome
```

The existing immutable Execution/Attempt/Workspace/Authority supplies Execution,
attempt, repository identity, source ref, base and human grant correlation. The
new nested capture freezes the absolute worktree path and its own generation.
The canonical store requires a lease for capture changes, append-only phases,
immutable identity/candidate/observations and independent snapshot copies.
No additional store, journal, ref ledger, public status/verb or Workflow schema
is introduced. The nested capture has explicit version 1; absent capture retains
old-record encoding and behavior, including the existing bounded codec.

```text
runner lost -> reopen canonical store -> claim later generation
  -> existing workspace/integration uncertainty blocks successors
  -> observe frozen worktree + source HEAD + index/status digests + candidate
       exact candidate, single recorded base parent, clean index/tree -> captured
       source observed at frozen base                            -> refused
       missing/unreadable/foreign/mismatched facts                -> unknown
  -> append observation (or separate recovery observation if already observed)
  -> preserve history; no staging, new commit, ref advance, reset or cleanup
  -> human inspection/disposition; never automatically replay capture/integration
```

`refused` describes the source ref observed at base. It does **not** assert that
staging/object writes never happened. A known captured source does not establish
integration success. An interrupted prepared integration keeps its existing
unknown outcome even when capture was known. A later contradictory observation
never rewrites the earlier capture success.

## Deterministic failure matrix

The tests use real temporary local repositories/worktrees and the canonical
ExecutionStore, with no remotes or vendors. Crash boundaries are deterministic
injected stop/publication failures; runner loss closes the actual lease and
recovery reopens the actual store. They do not claim a machine power-cut test.

| Boundary/case | Executable coverage and assertion |
| --- | --- |
| Intent refused, target observation fails | `TestCaptureAdmissionRefusalHasNoGitEffects`; refs, index and object inventory unchanged. |
| Actual store refuses intent | `TestCaptureStorePublicationFailureRecovery/intent`; no Git mutation and no invented intent. |
| Cancel or lost fence before staging, after staging, after tree write, after object creation, after candidate registration, after source CAS | `TestCaptureCrashFenceRecoveryMatrix` (12 cases); exact path/generation/SHA, retained staged/unstaged/untracked file content, index/ref/object inventory unchanged during recovery/reentry, dependent work blocked. |
| Store refuses candidate | `TestCaptureStorePublicationFailureRecovery/candidate`; object may exist but candidate is not adopted and source does not advance. |
| Store refuses source observation or integration preparation | `TestCaptureStorePublicationFailureRecovery/observation` and `/prepared`; persisted candidate enables observation after reopen, without replay. |
| Integration outcome publication lost after target CAS | `TestCaptureLostIntegrationOutcomeDoesNotReplay`; prepared becomes unknown, target and objects retained, no successor/reentry effect. |
| Candidate persisted before source CAS; source/target/binding moved | `TestCaptureMovedRefsAndExactPersistedCandidate`; reload store in pre-CAS callback, verify exact candidate and source still at base; foreign source/binding refused and target CAS preserves foreign advance. |
| Source moved during interruption | `TestCaptureRecoveryMovedSourceIsUnknown`; foreign head is observed, never treated as writer output. |
| Previously observed capture success, then source drift before recovery | `TestCaptureRecoveryPreservesPriorSuccessAndObservesDrift`; original success preserved, separate unknown recovery observation. |
| Writer-owned clean commit / dirty leftovers | `TestCaptureOwnCommitAndExecutableConfiguration`; clean commit retained, leftovers not adopted. Existing Git-boundary tests retained. |
| Executable filter/diff configuration and hooks | Same test plus existing Git-boundary tests; filters/drivers refused and hook override disabled, sentinel never executed. |
| Store immutability, lease and failed-update isolation | `TestCaptureStoreHistoryAndSnapshotIsolation`, `TestCaptureAdmissionAndImmutableTransitions`; unfenced intent, combined intent/candidate and identity rewrites rejected. |
| Codec, exact integration correlation | `TestCaptureCodecAndCorrelation`; round trip, candidate immutability, prepared integration still requires the real non-base candidate. |
| Human result review | `TestCapturePreservesHumanResultReviewDigest`; capture does not alter the human-approved writer-result digest; earlier nil-capture digest bytes stay unchanged. |
| HOME-aware real post-merge check | `TestIntegrateUncommittedSandboxedWriterWithHomeAwareCheck`; now uses an enrolled validated attempt and a real command. |

Targeted capture matrix: PASS (Go 1.27.3, Linux arm64, exit 0;
`go test -count=1 ./internal/stageexecution -run 'TestCapture|TestIntegrateUncommitted'`).
Writer-result review unit: PASS (Go 1.26.0, exit 0;
`go test -count=1 ./internal/workflow -run TestCapture`).
Final validation: all rows below PASS, observed exit 0. Build/test/vet/module checks and Staticcheck used the already installed
Go 1.26.0 toolchain with matching `GOROOT`/`PATH` and `GOMAXPROCS=2`;
`-p` limits package concurrency, not test coverage. Formatting used ambient
Go 1.27.3 gofmt.

| Command | Result |
| --- | --- |
| `go test -count=1 -p 2 ./...` | PASS; every package, including `cmd/lingo` (493.128s), compatibility, local store, and stage execution (82.151s). |
| `go test -count=1 -p 1 ./internal/workflow ./internal/stageexecution` | PASS; final focused repetition after the human-review digest correction (workflow 2.572s; stage execution 197.833s). |
| `go test -count=1 ./internal/gitworkspace` | PASS; final hardened status boundary (2.980s). |
| `go vet -p 2 ./...` | PASS. |
| `go build -p 2 ./...` | PASS. |
| `go mod verify` | PASS; all modules verified. |
| `./scripts/check-go-quality.sh format` | PASS. |
| `./scripts/check-go-quality.sh tidy` | PASS; `go mod tidy -diff`, no module edits. |
| `STATICCHECK=/home/grokbot/go/bin/staticcheck ./scripts/check-go-quality.sh staticcheck` | PASS; pinned Staticcheck 2026.2.1 (0.8.1). |
| `./scripts/test-docs-offline.sh` | PASS; Linux arm64, 26 corpus files with closed links; tampering refusals observed. Its Runtime/network/source-denial qualification remains PARTIAL/NOT_RUN. |
| `./scripts/validate-repository.sh .` | PASS; bootstrap, security, documentation and ADR governance checks. Optional native Runtime behavioral scenarios not requested. |
| `git diff --check` and `git diff --cached --check` | PASS. |
| `./scripts/check-sensitive-files.sh --staged .` | PASS; staged paths/content reviewed. Gitleaks unavailable (NOT_RUN); no scanner installed for this correction. |

Earlier development runs are not delivery passes: one concurrent package run
failed the existing 3-second persisted-cancel wait; the final focused and full
runs passed without changing that test or its timeout. Superseded validation
processes were stopped when their source snapshot changed. Initial compile
errors were corrected and covered by later runs. Staticcheck was initially
absent from PATH and then incompatible with Go 1.27 export data; the installed
pinned binary passes with the matching Go 1.26 toolchain/root. Offline docs
initially rejected a source-only relative link; it now uses the canonical branch
URL and the offline test passes. An obsolete test helper was removed after the
Staticcheck diagnostic. No test assertion or production gate was relaxed.

## Local technical decisions

| ID | Decision, alternative and verification |
| --- | --- |
| TD-C399-01 | Optional versioned capture nested on Attempt reuses its immutable Workspace/Authority and the existing store. Duplicating correlation in a second store/ref ledger was excluded by the approved decision. Codec/store tests cover the new facts. |
| TD-C399-02 | Separate original and recovery observations preserve prior evidence. A known capture never authorizes automatic continuation after runner loss; tests reopen the store and assert no replay. |
| TD-C399-03 | Separate durable callbacks from Git effects; fenced store checkpoints and exact worktree/config validation precede each mutation. Existing source/target CAS remains mandatory. Fault tests cover each boundary. |
| TD-C399-04 | Exclude principal-owned capture facts from the writer-result digest, retaining all existing writer-result fields. Otherwise adding capture would invalidate a real human review. A required-review test verifies unchanged digest and valid integration preparation; old nil fields serialize identically. |

Documentation reconciliation was delegated as an independent bounded unit using
a bounded document capability with explicit medium reasoning; its changes were
inspected locally and covered by repository/offline-doc validation. A bounded
read-only review checked capture/store/Git invariants; executable validation is
reported separately and is the stronger evidence. No delegated external writes.

## Limits

- A crash after object creation but before candidate persistence may leave an
  unreachable object. It is retained; no candidate is reconstructed from it.
- The bounded index/status digests are forensic observations, not authority or
  backups. Retained Git objects, files and the canonical Execution remain needed
  for human inspection. Observation failure remains unknown.
- No automatic recovery continuation or new public resolution command. Existing
  workspace/integration recovery restrictions remain; a human must inspect and
  choose an authorized disposition. No retroactive attribution of historical
  principal-created commits without a capture record.
- Local lease checkpoints precede Git operations; filesystem/Git and the store
  are not one atomic transaction. Persisted intent/candidate and conservative
  observation handle interruption between them. No claim of hostile filesystem
  isolation, host power-loss qualification or remote-provider transactionality.
- Paid/vendor inference and remote-host E2E: NOT_RUN; no separate grant. Historical
  vendor evidence does not validate this protocol. Runtime harness behavioral
  scenarios are NOT_RUN unless explicitly requested by the repository validator.
- No merge, release, deploy, AXM-12/P-AC3 acceptance or card closure.
