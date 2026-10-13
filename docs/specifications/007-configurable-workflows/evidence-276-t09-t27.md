# I276-T09 / I276-T27 implementation record

Scope: [T09](plan-tasks-276-stage-agent-execution.md#i276-t09--276-admission-guards-for-versioned-contracts) and [T27](plan-tasks-276-stage-agent-execution.md#i276-t27--detached-bounded-supervisor), Plan §§3.2–3.3, 3.5–3.8, 3.10, amendment A6, amendment-306 §3, and ADR-0026 decision 5. Implemented under the maintainer's explicit task request. This record does not change approval statuses, historical contracts, or claim operational Runtime evidence.

## Implementation boundary

- `stageexecution.Admit` centralizes operation classification and version admission. It consumes existing reader-validated Workflow/Execution/Profile snapshots and the canonical GraphDocument codec. Historical Graph1/2 inspection preserves bytes/digests; mutations refuse them. Legacy Executions, schema-1 Workflows regardless of effort, local-v1 configuration and incomplete migration retain their existing refusal categories. Admission grants no authority or capability and performs no migration/resolution.
- `Supervisor.Run` and `Launch` enforce admission and exact protected dispatch-authority binding before lease acquisition, preparation or detached spawn. The supervisor reloads under the nonblocking lease, refuses running/stale attempts, selects bounded nonconflicting first-attempt batches through the existing successor consumer, and excludes integration owners and retries.
- Each attempt receives fresh frozen-binding/authentication/effort-source checks and a compiled stdin packet through `CompiledSupervisorPreparation`, reusing the delivered graphapplication/runtimeadapter and coordination ingestor. Working invocation permissions derive from the persisted effect envelope. Ingestion runs while children execute; final drain, classification and terminal CAS occur independently for each attempt. Failed saves preserve recovery state.
- `LocalSupervisorStore` in the `cmd/lingo` composition root implements the stageexecution supervisor interfaces and delegates protected authority loads, lease acquisition and epoch-fenced CAS to the existing local stores. It requires trusted stage composition to supply reader-validated contract snapshots and canonical records.
- `OSProcessRunner.RunObserved` retains existing process-group/job containment and bounded stop confirmation, adding an initial process-correlation/save callback. Observation or initial-save failure requests bounded stop and cannot yield success. Historical `Run` behavior uses the same path without a callback.
- The same-binary launcher disconnects stdin/stdout/stderr from the initiating process and creates a separate POSIX session or Windows detached process group. Unsupported platforms fail closed. Correlation is returned to the owning stage-run composition; Windows supervisor-session identity is distinct from child job identity.
- `__stage-supervisor` is hidden and accepts only graph/project/preview/authority identities. There is no public command, arbitrary executable selector, retry/reconcile/integration port or authority flag. The binary's default entrypoint currently has no stage-run loader and refuses invocation. T25/T26 own publication/material reconstruction and installation of that loader, including the single-stage ledger composition; this internal graph supervisor does not claim the public dispatch journey is complete.

No new codec, dependency, durable script/workflow, historical fixture edit, Workflow/Execution migration or Provider operation was introduced.

## Test source

All new tests use fake runners, preparation, authentication, effort inspection, stores, leases and detachers; platform attribute tests inspect inert commands without starting them. Fixtures are existing synthetic graph vectors. Temporary paths remain under test-owned roots; tests never read/write HOME, invoke vendor runtimes/probes, use network/inference or perform Provider effects.

- The admission matrix names all 15 historical rows, with owner-boundary checks for #274 sequential inspection and next-gate handoff for capability/auth/authority checks. Additional cases cover Graph2 mutation, unsupported/missing successor versions/fields, absent frozen tuple, incomplete migration and default-deny unknown operations.
- Historical Graph1/2 mutations across run/retry/resume/cancel/reconcile/integrate are refused; inspection preserves exact encoded bytes and SHA-256 digests.
- Two concurrent fake children complete through independent terminal saves while the integration owner remains pending. Restart does not redispatch completed work; stale running attempts require recovery.
- Failure injection covers lease contention, canonical record loading, preparation, first/second initial save, live ingestion, final drain, terminal save and drift between children. Context interruptions at each save boundary model supervisor loss without launching/killing a real process.
- Missing committed authority, forged reference, foreign graph, wrong preview/scope and legacy version guards refuse before any child or detached spawn.
- Fake cancellation before dispatch and during work stops further attempts; confirmed stop gets the persisted cancellation reason. Initiating-context loss does not own the independent supervisor run.
- Compiled preparation tests exercise frozen-tuple drift, wrong child/epoch/inbox, effort conflict, authentication failure and missing composition without any external observation.
- Linux/Darwin and Windows detachment flags are tested through inert commands; other platforms have an explicit unsupported-path test.

## Verification

- `gofmt` applied; diff whitespace and format checked.
- Offline Go 1.26 `go vet` passed for `internal/stageexecution`, `internal/executiongraph` and `cmd/lingo`, including test-source analysis, using the already available toolchain/module cache under `/tmp`, with network access disabled and caches outside HOME.
- Offline cross-platform `go vet` also passed for Windows/amd64 and Darwin/amd64. An additional FreeBSD/amd64 check cannot compile the existing `internal/local` package because its platform adapters do not define `renameNoReplace` and `checkPrivateACL`; that platform remains unverified and was not broadened in this change.
- No `go test` command was run. All new test behavior, actual process detachment/survival and platform failure-injection behavior remain unexecuted. No real Runtime/probe/inference/Provider effects or commits were performed.

Manual review covered version default-deny, frozen-tuple anti-fallback, exact authority scope, lease-before-CAS ordering, process uncertainty, cancellation races, per-attempt final save, no retry/integration dispatch, test isolation and historical artifact preservation. Public stage-run composition remains explicitly outside these task changes.

### Import-cycle correction

Moved the local-backed supervisor adapter to `cmd/lingo/supervisor_store.go` after placing it in `internal/local` created a test import cycle through workflowcompiler. The composition root imports both local and stageexecution; neither production package imports the other. Adapter behavior and existing fake tests are unchanged, with compile-time assertions for the store and lease interfaces.

Offline Go 1.26 `go vet` passed for `internal/local`, `internal/stageexecution`, `internal/executiongraph` and `cmd/lingo`, including test-source analysis. `git diff --check` passed. Both architecture-checker profiles were run: domain failed on the existing `encoding/hex` import and application failed on the existing `strconv` import in untouched packages; those unrelated allowlist failures remain unresolved. No `go test`, real runtime/provider operation or commit was performed for this correction.

The composition-root relocation was checked with `gofmt`, `git diff --check` and static import inspection. No `go test` was run for this relocation; existing fake tests were preserved unchanged.
