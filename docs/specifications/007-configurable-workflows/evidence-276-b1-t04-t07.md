# #276 B1 — I276-T04 and I276-T07 implementation evidence

## Scope and authority

Implemented under the explicit maintainer instruction on
`feat/276-stage-agent-execution`, continuing the uncommitted
`internal/executiongraph/dispatch_authority.go`. Inputs include the full task
plan, ADR-0026, the successor amendment/field appendix, ADR-0007, the existing
T01/T03/T06/T08 implementation and the five preceding commits. Recorded ADR
and Task approval statuses are unchanged. No network, real probes, inference,
commit or publication was performed.

The implementation units were delegated separately for T04 and T07; the main
session integrated inventory/corpus-writer coverage, documentation and checks.
A separate engineering/security review identified and corrected single-stage
scope reset, stale ledger read-back, orphan receipt re-apply, exact-wire digest
verification and equal prior/new ledger generation issues.

## Acceptance mapping

| Requirement | Implementation and executable evidence |
|---|---|
| I-1: terminal classification required | `LoadSuccessorConsumer` uses the T01 validator; consumer tests reject missing reason/qualification. No duplicate codec or classifier is added. |
| I-2 / I-3: resolvable result/blocker | Load validates canonical accepted record projections with exact parent/child/revision/attempt/digest, kind and outcome. Missing and foreign references are refused. |
| I-4: persisted consumers | The immutable loaded snapshot supplies readiness, status, dispatch-batch guards and integration preview. Consumers never call `ClassifyAttempt`. Restart/state matrix tests compare persisted outcomes. |
| I-5: late records cannot rewrite status | Consumer tests supply a later blocker without changing the persisted successful result. Copies protect the snapshot from caller mutations; T01 update validation continues to freeze terminal attempts. |
| I-6: historical/unmarked mutation refused | The successor consumer refuses actual decoded Graph1/2 documents and missing policy markers. Existing historical APIs/readers retain their prior semantics. |
| Retry and dependencies | Failed/blocked/cancelled retries require explicit authority and remaining attempts; unknown/ambiguous needs uncontradicted `retry_safe`. Running and skipped cannot dispatch. Facts never promote success or unlock dependents. Persisted cancellation blocks new attempts. |
| Integration eligibility | `result_recorded` and the exact qualifying canonical result projection are required. Caller result mismatch, foreign digest and `result_missing` are refused. Reconciled integration preview binds the actual prior attempt count. Integration execution/roll-up remains T33. |
| G16 concurrency | Proposed batches count already running attempts against recorded `stageConcurrency`, reject duplicates/unready children and conflicting effects, including effects of running children. |
| Protected T07 record and linkage | Closed bounded immutable format-1 receipts bind preview, parent/revision, proposal, effects, operator/time and scope. Graph authority reference and every envelope reference must match. `CheckFirstDispatch` refuses retries, started children, integration and cancelled graphs. |
| Graph / single commit | `GraphStore.CommitDispatch` holds the run lease before graph locks. `WorkflowStore.CommitSingleDispatch` binds a distinct Execution3 ledger generation plus single stage/ordinal/attempt. No Execution format changes. Equal-byte updates are refused before mutation. Single-stage scope cannot be renewed by changing the proposal, attempt or ledger generation. |
| Single commit point / recovery | The existing protected publication marker additionally names the immutable receipt and exact digest. Receipt publication precedes canonical graph/ledger rename, with precommit and postcommit read-back. Intermediate readers fail closed. Recovery accepts the new generation only with the bound receipt; absent/inconsistent committed receipts are preserved for review. Exact inert receipt re-apply is allowed; changed single-stage authority is conservatively refused. |
| Fault injection | Graph and actual Execution3 ledger unit publication tests cover all 18 F0–F8 windows: F0/F1 have no pending generation, F2 unmarked preparation is preserved, F3–F5 restore prior, F6–F8 finalize committed. Tests also cover missing receipts, precommit re-apply and refusal of commit after receipt removal. |
| Inventory / compatibility | Additive `dispatch_authority` kind, exact family/project/content-addressed location, canonical load/relocation tests and real-writer coverage. Existing frozen snapshots are untouched. Pure inventory tests reject renamed, foreign-project, wrong-family and reformatted receipts. |

Canonical projections are a trusted storage read boundary, not arbitrary
agent assertions. T04 does not add ingestion, supervision or public CLI entrypoints.
T07 does not compute preview digests, adopt an Execution codec, or grant retry,
cancel, reconciliation, integration or capability-probe authority.

## Executed validation

Go commands used `GOPROXY=off GOCACHE=/tmp/axiom-276-go-cache` to prevent
downloads and keep the cache in writable temporary storage.

| Check | Result |
|---|---|
| `go build ./...` | Passed. |
| `go vet ./...` | Passed. |
| `go test ./internal/executiongraph -count=1` | Passed, including existing historical/successor codec, scheduler, integration and new consumer/authority tests. |
| `go test -race ./internal/executiongraph -count=1` | Passed. |
| `go test -race ./internal/executiongraph ./internal/local -run 'Test(Consumer\|DispatchAuthority\|DispatchCompanion\|EveryStateRootWriterDirectoryIsInventoried)' -count=1` | Passed; no race reported. |
| Public graph/single-store tests | Attempted; both stop before assertions with `unsafe project storage`, at private-root ancestry validation in this sandbox. |
| Inventory canonical load / relocation tests | Attempted; writer setup stops at `unsafe_source`, before inventory assertions. |
| `TestStableCorpusIsAppendOnly` | Passed. |
| `TestStableCorpusResolvesToDirectUpgrade` | Attempted; sandbox ancestry causes `unrecognized_or_unsafe_content` at the roots of the existing v0.10.0 snapshot. |
| Corpus freeze with `AXIOM_FREEZE_STATE_CORPUS=v0.12.0-issue276.b1t07` | Attempted through `TestEveryV1WriterIsRecognizedByInventory`; setup fails at `unsafe_source`. No snapshot or manifest was written. |
| Windows amd64 / Darwin arm64 cross-compilation | `go test ./internal/executiongraph ./internal/local -run '^$' -exec /bin/true` with each GOOS/GOARCH passed. No native tests executed. |
| `TestSingleDispatchRetryLedgerGeneration` | Passed: valid distinct Execution3 retry fixture preserves the frozen binding and stage scope; public scope/refusal tests also include no-op and changed proposal/attempt cases. |
| `gofmt` / `git diff --check` | Clean. |

## Remaining validation

Public store, inventory writer/load/relocation and direct-upgrade tests must run
outside this sandbox, with the repository's private-storage checks intact. Then
freeze the actual writer output using the same `v0.12.0-issue276.b1t07` label
(still unused), review MANIFEST/PROVENANCE and rerun append-only/direct-upgrade
guards. The authority kind remains additive pending that freeze. No synthetic
snapshot was fabricated to replace a failed real-writer freeze.

Native Windows/Darwin execution remains unverified; cross-compilation alone
does not prove native protected-filesystem or recovery behavior. No blocking
correctness finding remained in the scoped static review, but the unexecuted
storage assertions and corpus freeze remain required validation evidence.
