# #276 B1 — I276-T06 and I276-T08 implementation evidence

## Scope and authority

Implemented under the explicit maintainer instruction for T06 and T08 on
`feat/276-stage-agent-execution`. This evidence does not change the recorded
acceptance status of ADR-0026, the successor amendment or the Task catalog.
No Runtime/provider inference, real capability probe, external network access,
commit, push or corpus freeze was performed. `.tmp-276` was left untouched.

Inputs: complete T06/T08 sections, Plan §§3.2, 3.5, 3.7, 3.8 and relevant §9
coverage, constitution, Spec 007/amendment #276, successor amendment and T01
field appendix, ADR-0007, ADR-0009, ADR-0025 and ADR-0026. Existing T01 codec,
shared OS directory locks, anchored private filesystem access, ADR-0007
publication/recovery and T43 process observation are reused.

## Implementation and acceptance mapping

| Requirement | Implementation and test evidence |
|---|---|
| One non-blocking per-graph lease; first in lock order | `RunLeaseStore.Acquire`/`Probe` use the existing exclusive non-blocking OS directory lock on `run-leases/v1/<project>/<parent-hash>.json/`. Successor saves require the held lease before graph CAS locks. OS contention tests use concurrent goroutines and a separate Go test process. |
| Monotonic durable epochs | Bounded strict format-1 `epoch.json`, shared protected publication/read-back, increment on acquisition, graph-derived minimum epoch, overflow refusal, and rejection of a persisted epoch below the supplied floor. Acquire/release/restart tests check increasing epochs. |
| Fencing and replacement refusal | Held lease validation checks its entire directory anchor, exact epoch-file object and persisted epoch. Successor CAS/update validation freezes existing attempt correlation and fences new attempts/terminal writes to the current epoch. Foreign-state-root leases are refused, and saves pin the handle against concurrent Close through the commit point. The lease is checked again immediately before commit. Replacement, symlink, lower-epoch and commit-drift tests are included. |
| Deterministic stale/orphan observations | `ObserveAttemptHealth` retains persisted status: running/free lease is stale; exact PID/start/group match is orphan. Reused PID/start mismatch is not an orphan; observation failures remain unknown. Linux uses boot ID/kernel start ticks; Darwin uses kernel process start identity/group; Windows uses kernel creation FILETIME/job leader correlation with observable job membership. Synthetic identity/health tests require no live Runtime. |
| Exact fact binding; append-only transaction | `AppendReconciliationFact` binds parent/child/revision/attempt, attempt epoch, observation digest, separate operation authority reference and actor/time using the T01 validator. Reusing the graph dispatch-authority reference as reconcile authority is refused. `GraphStore.AppendReconciliation` acquires the lease, rechecks supplied observation sources, and CAS-publishes only appended facts. The lease component is derived from successful acquisition, so an apply self-probe cannot be mistaken for a live supervisor. |
| Stale fact without status rewrite | T01 successor validation admits facts on stale running attempts, with fact time bounded by attempt start. Local apply refuses a live run holder. Running and terminal fields remain frozen; a running attempt is never redispatched. The field appendix records this T08 refinement. |
| Duplicate/contradictory facts | Semantically duplicate facts with distinct IDs remain present. Opposing classifications remain present and are surfaced by `ReconciliationDisposition`; contradictions/recovery-required facts deny retry. Prior fact prefixes cannot be rewritten/deleted. |
| Persisted scheduler/integration reads | `SuccessorRetryEligible`, `ReadySuccessorChildren` and `SuccessorIntegrationAttemptAvailable` validate successor state and consume persisted facts only. Explicit retry after unknown requires an uncontradicted `retry_safe` fact and remaining attempts. Facts never establish child/integration success or unlock dependencies. Historical scheduler/Graph1/2 behavior remains separate. These are foundation read paths for later successor consumers, not a new public command or supervisor. |
| Observation gathering | `GatherReconciliationObservations` accepts bounded lease/process/worktree/canonical-record/quarantined-inbox sources, validates digest projections, and retains process observation uncertainty. Quarantined content is never a qualifying result. No worktree mutation, inbox ingestion or provider call is added. |
| Inventory and recovery | `run_lease` is an additive kind, its directory is inventoried, `writeEveryV1Kind` exercises acquisition/release, and canonical-location/load tests cover it. Inventory admits successor graphs with the codec's 1 MiB bound. Recovery enumerates graph CAS and per-graph lease publication directories. Fault tests cover epoch publication F0–F8 and graph fact interruption before/after commit. The stable corpus is unchanged. |

## Executed validation

All Go commands used `GOFLAGS=-mod=mod GOPROXY=off`. After the default Go
cache proved unwritable, `GOCACHE=/tmp/axiom-276-go-cache` was used; this
changes no filesystem security control.

| Command/check | Result |
|---|---|
| `go test ./internal/executiongraph -count=1` | Passed: complete package, including historical/successor codec and new reconciliation tests. |
| `go test -race ./internal/executiongraph ./internal/local -run 'Test(AttemptHealth\|Reconciliation\|GatherReconciliation\|ProcessIdentity\|RunLeaseOSLock)' -count=1` | Passed: domain tests, OS lock contention and separate-process exclusion; no race reported. |
| `go test ./internal/local -run 'Test(RunLeaseOSLock\|EveryStateRootWriterDirectoryIsInventoried)' -count=1` | Passed. |
| `go vet ./internal/executiongraph ./internal/local` | Passed. |
| `go test ./... -run '^$'` | Passed: repository compilation only, no claim that all repository tests ran. |
| `GOOS=windows GOARCH=amd64 go test ./internal/executiongraph ./internal/local -run '^$' -exec /bin/true` | Cross-compilation passed; no Windows tests executed. |
| `GOOS=darwin GOARCH=arm64 go test ./internal/executiongraph ./internal/local -run '^$' -exec /bin/true` | Cross-compilation passed; no Darwin tests executed. |
| New private-store tests and inventory writer/canonical-store tests | Attempted; fail at the sandbox's private storage/source ancestry checks (`unsafe project storage` / `unsafe_source`), before their storage assertions, as anticipated by the maintainer. Must run outside this sandbox. |
| `gofmt` and `git diff --check` | Clean. |

The private-store suite includes acquire/release/restart, live probe, concurrent
acquisition, directory/epoch replacement and symlinks, lowered epoch, F0–F8
failures, recovery enumeration, stale/unknown facts, exact CAS/revision binding,
current-epoch terminal save, terminal rewrite refusal, commit-time fencing, foreign-state-root lease refusal, concurrent Close serialization and
fact publication crash windows. These tests are authored and compile; their
storage assertions are **not verified in this sandbox**.

## Review and remaining validation

No blocking finding identified in the static engineering/security review of
this scoped change. Checks addressed exact-target paths, non-blocking locks,
lease-before-CAS ordering, bounded strict records, frozen attempt/process/fact
state, truthful pre/post-commit recovery, PID reuse, uncertainty, duplicate and
contradictory facts, and absence of new dependencies/external calls. No ADR or
historical fixture was rewritten.

Native Windows/Darwin execution and private-store tests outside the sandbox
remain required. In particular, the existing T43 Windows job is unnamed:
after leader loss, a restarted observer cannot prove whole-job termination
without the held job handle. `ObserveRecordedProcess` therefore returns unknown
and blocks `retry_safe`, rather than claiming confirmed stop. Linux/Darwin also
retain uncertainty if group members remain without a matching leader identity.
Reconciliation authority is an operation-scoped reference supplied by the
trusted caller; this task does not implement T07 protected dispatch authority,
T27 supervision, T30 CLI or T31 workspace disposition/retry application.

## Changed files

- `internal/executiongraph/successor.go`: stale-running fact validation and frozen running-state updates.
- `internal/executiongraph/reconciliation.go`, `reconciliation_test.go`: health, observations, fact append/conflict handling, persisted eligibility/read paths and acceptance tests.
- `internal/executiongraph/process_identity_linux.go`, `process_identity_darwin.go`, `process_identity_windows.go`, `process_identity_other.go`, `process_identity_linux_test.go`: native identity observation, fail-closed fallback and synthetic kernel parsing tests.
- `internal/local/run_lease.go`, `run_lease_inventory.go`, `run_lease_test.go`: lease persistence, anchored validation, inventory and contention/fault/recovery tests.
- `internal/local/successor_graph_store.go`, `successor_graph_store_test.go`: successor CAS, epoch fencing and append-only fact transactions/tests.
- `internal/local/inventory.go`, `inventory_identity_test.go`, `recovery.go`, `state_compatibility_contract_test.go`: additive kind, successor recognition, canonical lookup/relocation, recovery enumeration and writer coverage.
- `docs/specifications/007-configurable-workflows/appendix-276-successor-graph-field-contract.md`: reconcile stale-running fact requiredness/time with T08.
- `docs/specifications/007-configurable-workflows/evidence-276-b1-t06-t08.md`: this report.
