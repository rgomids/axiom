# Integrated S2 reconciliation and S3 readiness

Attempt: `RECONCILE-S2-INTEGRATED-CODEX-01`, AXM-42, same Draft
[PR #366](https://github.com/rgomids/axiom/pull/366). Initial local/remote/PR SHA:
`08f2e0e42699b23d6d0f536fe0c77ddb6bb791a1`, clean worktree, verified before edits.
Authority: accepted [Tasks S2.1–S2.4](tasks.md), [Plan](plan.md) and
[ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md).
Cumulative S2 source comparison: `1bf0b6c..08f2e0e`, plus these bounded corrections.
This is an executor technical reconciliation, not an independent cumulative PR
approval, human acceptance or permission to implement S3. DESIGN: NOT_APPLICABLE
(CLI/domain only). No external-provider body or private installed state imported.

## Bounded findings

| ID | Disposition / evidence |
| --- | --- |
| m1 | RESOLVED: category G survival proof in [deletion report](deletion-s2-04.json), exact Evidence fields, five Coordination constants, two types and four coordination-path functions listed. `grep -R` across cmd/internal finds no production BuildEvidence caller or VerifyCoordination implementation. Existing acceptance/integration tests pass nil; no positive test verifier is live evidence. Historical format is preserved; comment no longer claims a production implementation. S3.4 owns deletion. |
| m2 | RESOLVED: `TestFrozenHistorySemanticValidation` uses frozen released bytes, standard JSON/SHA-256 resealing, and only DecodeStream/ValidRecord/ValidStream. Credentials, role markers, enum/whitespace, empty/oversize/control values, invalid usage source/unit/value/status refuse with ErrInvalidRecord; ordinary keyword prose and supported statuses pass. Digest tampering has the same error classification. Every frozen stream remains readable without rewriting bytes. No operational API restored. |
| n1 | RESOLVED: stageError says publication is retired, messages use private engine ingress, inspection uses workflow coordination list. Adjacent record-not-found advice names Execution messages. `TestPublishRefusesAbsentRunningContext` asserts the refusal and next-action. |

## Integrated acceptance evidence matrix

Source package paths below are under internal/ unless prefixed cmd/; tests live beside their source.
Results are finalized in the validation table and exact-SHA CI receipt. Stub labels
Codex/Claude establish common adapter behavior, never live vendor execution.

| Criterion | Source / authority | Focused negative or invariant test | Result / limit |
| --- | --- | --- | --- |
| S2.1 same loop/codec for one/many agents; serialized completions, early dependency progress | stageexecution/scheduler.go, engine.go; workflow/execution_transitions.go | TestEngineParallelReadyTasksStreamAndCommitWithoutLoss (A/B barrier, C before B finishes, writer concurrency=1, final revision) | PASS; bounded local fixture |
| S2.1 invalid dependencies refuse before spawn | workflow definition validation, scheduler_test.go | TestEngineParallelDependencyDefinitionsRefuseBeforeSpawn | PASS; cycle/missing agent/stage dependencies |
| S2.1 human gate, failure, cancel, unknown and stale fence stop successors | workflow/execution_transitions.go, stageexecution/scheduler.go | TestEngineParallelStopsDependentWork; TestExecutionParallelFailureStopsReadyWhilePeerStillRunning | PASS; independent terminal observations retained |
| S2.2 common supervised bidirectional exchange; validated result alone unblocks | stageexecution/runtime_exchange.go, exchange.go, engine.go | TestExchangeCodexClaudeBidirectionalUnblocksOnlyOnValidatedResult; TestExchangeMessagesCannotUnblockDependency | PASS; deterministic adapters, no vendor inference |
| S2.2 wrong attempt/runtime/forged sender/stale fence, duplicate/conflicting reply and late answer | workflow/execution_messages.go | TestTwoStubQuestionAnswerIngress; TestLateOrUnmatchedAnswerIsRejected; TestRuntimeExchangeAreaSafety | PASS; private rooted areas, same-UID access is not confinement |
| S22-CR-01 transport faults must not become success | exchange.go final drain; engine.go; execution_transitions.go | TestExchangeFailureIsSurfacedOnTheAttempt; TestMessagePumpFaults | PASS; unsafe send, persistent ingress and conflicting delivery fail attempt; transient ingress retried; dependent not spawned on fault |
| S2.3 isolated worktrees, candidate/base/target identity, dirty/conflict/shared-target race | gitworkspace/integration.go; stageexecution/integration.go; workflow/execution_integration.go | TestIntegrateParallelWorktreesSequentiallyWithCheckEvidence; TestIntegrateSameFileConflictIsExplicit; TestIntegrateStaleBaseAndDirectTargetDriftRefuse; TestIntegrateDirtyCandidateAndCheckedOutDestinationRefuse; TestIntegrateConcurrentRaceOnSharedDestination | PASS; temporary repositories, library composition only |
| S2.3 integrated only after merged-tree checks and exact final SHA/CAS | same source | TestIntegrateFailedPostMergeCheckStaysNotIntegrated; TestIntegrateUnknownGitOutcomeIsNeverRetried; TestIntegrateLostOutcomeBecomesUnknownForNextRunner | PASS; no remote push or automatic unknown replay |
| S23-CR-01/02 unknown precedence and persisted cancellation during/between/start checks | integration.go runIntegrationCheck / watcher; workflow/execution_integration.go | TestIntegrateUnknownCheckBlocksReplay; TestIntegrationCheckObservationSemantics; TestIntegratePersistedCancelChecks; TestIntegrationUnknownCheckCodecAndDisposition | PASS; unconfirmed stop stays unknown; no next check/Advance |
| S23-CR-03 append-only integration intent/terminal evidence, snapshot isolation | local/execution_store.go preservesExecutionHistory / clone | TestExecutionStoreRejectsIntegrationHistoryRewrite; TestExecutionStoreUpdateIntegrationSnapshotIsolation; TestExecutionStoreIntegrationPreparedAndHumanDisposition | PASS; rejected Save leaves revision/history unchanged; DF-RR-01 remains bounded |
| Exact repository/effect permissions/target status | workflow/execution_integration.go; cmd/lingo/execution_run.go; local/execution_store.go | TestIntegrateRepositoryBindingAndUnauthorizedSources; TestIntegrateInvalidCandidateAndEscapedWorktree; TestIntegrateHumanGateAndCancelProhibitEffect; TestExecutionDispatchRechecksProjectAdmissionBeforeSpawn; TestExecutionDispatchRefusesPersistedRebindBetweenStages; TestCanonicalReadinessIgnoresLegacyLedgersAndGrantsNoEffects | PASS; local status reads grant no delivery authority |
| S2.4 one durable ingress/schema, four kinds, forged authority inert | workflow/execution_messages.go AcceptMessage; engine.go Ingest; exchange.go pump | TestFourMessageKindsPreserveAllOperationalTruth; TestIntegrationMessagesGrantNothing; TestHistoryExportsAreReadOnly | PASS; question/answer/blocker/result only; engine entry paths share AcceptMessage/fence/writer |
| S2.4 history refusal/retention, no shadow L/E store/writer/codec/coordinator | local/inventory.go; coordination/coordination.go; CLI stage_service.go | TestRetiredCoordinationWithoutGraphRefusesAndNeverRewrites; TestEveryV1WriterIsRecognizedByInventory; TestExecutionStoreRefusesLegacyBytesInNewNamespace; frozen history tests | PASS; inventory-only decoder, no conversion or historical dispatch |
| S2.4 installed CLI/help/skill/docs and reduction/import report | internal/cli/workflow_stage.go; docs/commands.md; docs/operational/workflow.md; bundled axiom-work-item skill; deletion report | TestEveryPublishedSharedRevisionStaysOwned; installed docs journey; exact Git recount | PASS subject to validation below; public publish always refuses, legacy cursors refuse |

## Review receipts and correction closure

Read native GitHub review bodies/state/commit IDs through API, plus Linear receipts:
[S2.2 #5742024731](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5742024731)
at `95cfa3d2dcc5458af7cd9653b52a8c49e8bb59ff`,
[S2.3 #5744033110](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5744033110)
at `4de87d18a098db7518bd52d69efd159a68c246c5`,
[S2.4 #5744932374](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5744932374)
at the pinned initial SHA. All are COMMENTED, analytically favorable only for
their bounded scopes. Their earlier RED reproductions are inherited evidence,
not rerun here. S22-CR-01 and S23-CR-01/02/03 are covered again above; S2.3's
stale unknown-check documentation was corrected in S2.4 and remains accurate.
No slice review is presented as cumulative S2/PR approval.

## Inventory, reduction and ownership

The [S2.4 report](deletion-s2-04.json) retains its original measurement scope;
its before/after totals were independently reproduced from Git blobs at
`4de87d1` and `08f2e0e`. Production: 307/66,703 lines → 305/66,081;
tests: 355/74,706 → 350/73,438. Exported lexical declarations: production
types 774→763, functions/methods 972→956; test functions 1883→1854.
The 24 changed/deleted Go paths exactly match that incremental diff.

After this reconciliation: 305 production files / 66,079 physical lines /
763 exported types / 956 exported functions; 350 test files / 73,555 lines /
1855 exported test functions. Corrections add safety coverage, not production
APIs. These are all cmd/internal Go files, including replacements; not the
final S3 candidate-envelope savings. No target is satisfied by deleting tests.

Cumulative S2 retired paths include coordination inbox.go/ingest.go/read.go,
local attempt_inbox.go/coordination_ingest.go/coordination_store.go,
runtimeadapter/packet.go and their exclusively retired tests. S2.4's exact
removed-path list remains in the machine report. Retired writers include
Service.Publish, CoordinationStore.Publish/Latest/ReadStream, encoders,
AcceptanceVerifier.VerifyCoordination and TaskPacket preparation. No new writes
to coordination/v1; Record/Stream format 1 remain historical recognition only.

Only production import of coordination is local/inventory.go. New operations
use Execution4 in executions/v2, local.ExecutionStore, stageexecution.Engine,
workflow.Message/AcceptMessage; the launcher files are transport payloads, not
another durable truth ledger. executiongraph imports in the canonical launcher
reuse observed process primitives, not GraphStore or a Graph coordinator.
Historical G evidence/gate contracts and I delivery contracts still exist for
S3.4 retirement; they are not claimed deleted. No replacement L/E writer,
codec/store or coordinator was introduced. Raw historical bytes remain inert;
no private state, migration or cutover was exercised. graphify-out excluded.

## deferred_findings[]

| Stable ID | Status / reason | Proposed stage or trigger |
| --- | --- | --- |
| m1 / m2 / n1 | RESOLVED as above | This reconciliation |
| DF-S22-01 | RESOLVED by S2.4 obsolete task packet deletion | No additional work |
| DF-S22-02 | DEFERRED: Spec 007 historical links identify retired source; historical text is not current implementation | S3.4 documentation review |
| DF-S22-03 | DEFERRED: whole send-area reread bounded to 256 entries | Measure before raising bounds; S3.5 risk review |
| DF-S22-04 | DEFERRED: same-UID peer-area access remains possible; area attribution is not OS isolation | S3.5/pre-merge explicit limitation |
| DF-S23-01 | DEFERRED: public Integrate/worktree composition absent; launcher retains shared Workspace | S3.1 views and S3.4 public-contract reconciliation; no new CLI scope inferred |
| DF-S23-02 | DEFERRED: same-UID child ref writes detected, not prevented | S3.5/pre-merge security review |
| DF-S23-03 | DEFERRED: check-process identity not persisted across runner loss; unknown blocks replay | S3.5 recovery review; no native confinement claim |
| DF-S23-04 | DEFERRED: worktree disposal explicit; automatic deletion would expand effects | S3.5 operational handoff |
| DF-S23-05 | DEFERRED: Git >=2.38 required; older Git fails closed | S3.5 supported-environment qualification |
| DF-S23-06 | DEFERRED: pre-existing Git subprocess containment limits | S3.5/pre-merge security review |
| DF-S23-07 | DEFERRED: integration disposition/status lacks public composition | S3.1 views / S3.4 CLI reconciliation within accepted scope |
| DF-RR-01 | DEFERRED: store guard permits prepared→terminal without differentiating lease-free Save; no production lease-free integration-finishing caller | S3.4 caller/deletion closure, S3.5 review |

Earlier S1/S2.1 optional findings remain explicitly inherited from
[implementation-s2-01.md](implementation-s2-01.md#authority-and-s1-gate), not
closed or promoted to requirements. All outstanding optional findings must be
highlighted at pre-merge. None grants scope expansion or acceptance.

## technical_decisions[]

- TD-REC-01 (`scope_impact: none`): retain inert category G verifier rather than
  delete/change historical evidence semantics in S2.4. Call graph shows no
  runtime owner; S3.4 is the accepted deletion stage. Alternative deletion now
  adds format risk without improving operational ownership.
- TD-REC-02 (`scope_impact: none`): test semantic history validation by resealing
  frozen record copies with standard JSON/SHA-256. Alternative retired encoder
  reuse would restore operational API; digest-only tampering misses semantics.
  Usage values remain the released bounded string contract, not a new numeric
  parser; invalid range means byte limits and invalid shape/status.
- TD-REC-03 (`scope_impact: none`): preserve original S2.4 measurement snapshot
  and report current whole-tree counts separately. This keeps historical
  evidence reproducible and includes added tests/comment changes honestly.

## Validation and readiness

All local required checks completed on the correction tree (commit
`4458b04ffec578e95138f5f03a88a54c6757e106`); subsequent changes are this Evidence
and its index link. Exact final remote SHA/CI belong to the Linear completion
receipt, avoiding a self-referential committed hash.

| Command / evidence | Result |
| --- | --- |
| `go test ./internal/coordination -count=1 -coverprofile=/tmp/axm42-history-cover.out`; `go tool cover -func=...` | PASS; 87.8% package, validValue/validUsage/validResultOutcome each 100%; earlier 68.9% is historical baseline, not a threshold |
| `go test ./cmd/lingo -run '^TestPublishRefusesAbsentRunningContext$' -count=1` | PASS |
| Focused command below, `-count=1 -timeout=10m` | PASS; stageexecution 112.724s, workflow 1.005s, local 18.022s, executiongraph 0.628s, cmd/lingo 21.717s |
| Final `go test ./...` | PASS exit 0; unchanged passing packages may use Go cache; CLI 420.065s, stageexecution 108.105s, workflow 97.271s |
| `go build ./...`; `go vet ./...` | PASS |
| `go run ./scripts/check-architecture.go domain` and `application` | PASS; 12 domain / 46 application files |
| `./scripts/check-go-quality.sh all` | PASS, Go 1.26.0 + matching GOROOT, Staticcheck 2026.2.1 (v0.8.1), curated safety + U1000 set including tests; format/tidy included |
| `./scripts/validate-repository.sh .` final corrected run | PASS; repository/bootstrap, automation, ADR governance and sensitive-file checks; optional runtime behavioral scenarios NOT_RUN |
| `./scripts/test-docs-offline.sh` | PASS linux-arm64; tamper/refusal and installed corpus checks. Qualification PARTIAL: network/source denial and native Runtime observation NOT_RUN |
| Staged sensitive-file scan; `git diff --check` | PASS before commits |
| Prior exact-SHA CI, independently read via API | PASS: 41506205443 at 95cfa3d; 41529054802 at 4de87d1; 41537229210 and Delivery metadata 41537229204 at 08f2e0e |
| Race | NOT_RUN: CGO_ENABLED=0 and gcc unavailable; no race success claimed |
| gitleaks | NOT_RUN: unavailable; repository sensitive scanner passed |
| Model CLI battery, native vendor E2E, external live effects, real cutover, merge/release/acceptance | NOT_RUN by scope; no model fallback or subordinate agent |

Focused command (ordinary Go fixtures, no vendor CLI):

```sh
go test ./internal/stageexecution ./internal/workflow ./internal/local ./internal/executiongraph ./cmd/lingo -run 'TestEngineParallel|TestExchange|TestRuntimeExchangeAreaSafety|TestMessagePumpFaults|TestIntegrat|TestExecutionStoreRejectsIntegrationHistoryRewrite|TestExecutionStoreUpdateIntegrationSnapshotIsolation|TestExecutionStoreIntegrationPreparedAndHumanDisposition|TestFourMessageKinds|TestTwoStubQuestionAnswerIngress|TestLateOrUnmatchedAnswer|TestRetiredCoordination|TestEveryV1Writer|TestExecutionStoreRefusesLegacyBytes|TestExecutionDispatch|TestCanonicalReadiness|TestPublishRefusesAbsentRunningContext' -count=1 -timeout=10m
```

Earlier investigation failures remain FAIL, not relabeled: initial new diagnostic
test used a nonexistent NextAction field (twice, then corrected to
Completion.Next().String()); initial full suite/build-vet therefore failed.
The literal synthetic private-key marker triggered the repository scanner;
it is now assembled in memory without changing the refusal case or scanner.
The first broad focused run hit its global 180s deadline while a newly started
cancel subtest waited in filesystem fsync. Isolated
TestEngineParallelStopsDependentWork passed all six modes in 7.828s; the final
broader run uses 10m and passed. These corrections/failure investigations are
why affected checks and the full suite were rerun; no unexplained stress reruns.
The full corrected suite is the PASS above.

Current Codex quota read from account/rateLimits/read: ordinary usage allowed,
primary 37% used / 300 minutes; secondary 49% / 10080 minutes. No account
identifier, private state or raw response is committed. Claude quota NOT_RUN
(no fallback used); no claim that the user's prior snapshot is current.

Engineering/security self-review covered the cumulative S2 production delta,
its corrective guards, caller/import closure, supported CLI/docs and negative
assertions, following Axiom/Ponytail review discipline. No blocking finding
identified within accepted S2 scope. This executor review does not replace the
next independent reconciliation review.

### S3 readiness determination (after S2 technical PASS)

**S2 technical reconciliation: PASS. Recommendation: READY_FOR_S3_1.**
Tasks S3.1 depends on S2 complete; the criterion evidence and corrected findings
above establish technical conformance within the accepted S2 domain/library
scope. No unresolved S2 blocker was found. Prior favorable slice reviews are
inputs, not cumulative approval. Independent integrated review is still the
next gate; this recommendation authorizes no S3 implementation or dispatch.

S3.1 must use the same Execution owner and retain distinct repository/attempt/
artifact provenance, reject wrong-repository refs/paths and fabricated success,
and permit local completion without PR creation. Existing artifact/readiness
views are inputs to reconcile, not proof of S3.1 completion. No Provider or
infrastructure addition is needed or authorized by this readiness finding.

The accepted S3.2 boundary retains exact-effect push/PR authority, changed-target
invalidation and unknown/no-second-effect handling. S3.3 preserves rejected
history and linked rework without inherited permissions or agent acceptance.
S3.4 owns final G/I legacy DTO/store/codec/verb deletion, replacement-union
measurement and installed-contract reconciliation; minimum deletion budgets
remain unproven until then. S3.5 owns final technical/security reconciliation
and the later manual acceptance handoff. These are remaining tasks, not work
performed here. The deferred inventory above, unknown-process/cutover limits
and same-UID boundaries must remain explicit through pre-merge review.

No material architecture/product/design decision is requested. Remaining work:
independent S2 reconciliation review, separately authorized S3.1–S3.5, human
pre-merge/cutover disposition and later manual product acceptance. No native
runtime confinement, live vendor behavior or full AXM-42 completion is claimed.
No independently verified authorized continuation callback was established;
return handoff `AWAITING_EXECUTION_REPORT`, next gate
`INDEPENDENT_S2_RECONCILIATION_REVIEW_THEN_S3_READINESS`.

