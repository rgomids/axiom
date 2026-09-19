# POC — Lingo minimal Project lifecycle Evidence

## Delivered baseline

The merged POC baseline and this hardening change provide a local executable path:

```text
init -> validate -> reopen -> install -> reopen -> explicit name update
```

`init` creates a strict one-file `axiom.yaml`; `validate` is read-only;
`reopen` validates portable bytes and reports matching, absent, corrupt, or stale
ID-addressed local state; `install` writes only
`LINGO_STATE_ROOT/projects/<uuid>/installation.json`; `update` changes the name
through complete domain-state validation. JSON CLI events contain fixed categories
only. No command reads credentials or performs Git, network, process, Provider, or
Runtime actions.

## Reproducible checks

Run from repository root:

```bash
go test ./...
go vet ./...
go run ./scripts/check-project-domain.go
go run ./scripts/check-projectapp.go
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

The composed CLI test covers init/validate/reopen/install/update, verifies that
installation leaves portable bytes unchanged, and asserts one ID-addressed record.
Local adapter tests cover rejected symlink targets and refused unknown artifacts
before an update.

## Known limitations

- Only a minimal Project with one `axiom.yaml` is supported; documents/policies
  are refused rather than rewritten.
- The initial baseline used a root-local lock. The hardening change below adds
  directory locks, hard-link rejection and limited multiprocess proof. Full
  crash/ancestor-race proof remains unverified.
- Local bindings, credentials, Runtime observation, rename, Git, remote sync,
  provider integration and orchestration are unsupported.
- The original 2026-09-18 run lacked `gitleaks`; the 2026-09-19 E2E delivery ran
  gitleaks 8.30.1 successfully. Historical tool availability is not rewritten.

## Hardening and dogfooding — 2026-09-18

The bounded Axiom change exercised here is POC persistence hardening on
`agent/finish-poc` at implementation commits
`a704a6367485d3a800a36ee31521d9dd245b5b95` and
`f91caf0723d09ebfb47c15cf55ffd08fbdd465e8`. Run
`./scripts/dogfood-poc.sh` from the repository root.
It creates a temporary Project for that change, validates, reopens, installs,
updates its name, verifies portable bytes stay unchanged during install, verifies
one ID-addressed local record, and observes stale local state after update. Its
temporary files are removed after the run. The original local macOS run exited
0 and emitted the expected JSON categories for all eight commands. Draft PR
#29 additionally emits a structured `poc_dogfood` result with portable before/
after-install and after-update SHA-256 hashes plus the local-record hash.

| POC criterion | Executable evidence | Current result |
|---|---|---|
| Minimal lifecycle and failures | `go test ./cmd/lingo -run TestExecutableMinimalLifecycleAndFailurePaths -count=1` | Passed on macOS; real executable covers init, validate, reopen, install, update, no-op, conflict, missing input and stale state. |
| Portable/local separation | Black-box test and `./scripts/dogfood-poc.sh` | Passed on macOS; install preserves exact portable bytes and publishes one separate local record. |
| Create/update confinement | `go test ./internal/local -run TestPortableStore -count=1` | Passed on macOS for symlink ancestor, symlink Project, hard-linked manifest, unknown artifacts and interrupted temporary artifacts. |
| Process coordination | `go test ./internal/local -run TestPortableLockSerializesProcessesAndSurvivesCrash -count=1` | Passed on macOS; conflicting process cannot create while lock held and can proceed after owner death. |
| Create/update crash boundaries | `go test ./internal/local -run 'TestPortable(Create|Update)CrashBoundaryRequiresRecovery' -count=1` | Passed on macOS; subprocess termination before and after publication preserves old/new complete bytes and normal reads return `recovery_required`. |
| Local-install crash boundaries | `go test ./internal/local -run TestInstallationCrashBoundaryRequiresRecovery -count=1` | Passed on macOS; subprocess termination before/after record publication leaves absent/complete record and reopen returns `recovery_required`. |
| Local-record confinement | `go test ./cmd/lingo -run 'TestInstall(RejectsHardLinkedRecordWithoutWritingOutside|PreservesUnknownLocalArtifact)' -count=1` | Passed on macOS; install rejects a hard-linked record without changing its outside target, and install/reopen reject unknown local artifacts without deleting them or changing the record. |
| Pre-commit cancellation | `go test ./internal/local -run 'Test(PortableUpdateCancellationBeforePublicationPreservesOldBytes|InstallationCancellationBeforePublicationLeavesNoRecord)' -count=1` | Passed on macOS; cancellation at the final test barrier before publication preserves the old manifest or absent local record and removes owned attempt artifacts. |
| Permission denial before update | `go test ./internal/local -run TestPortableUpdatePermissionFailurePreservesOldBytes -count=1` | Passed under unprivileged macOS user; prior bytes stay intact. Root execution skips this case. |
| Read-only validation | `go test ./cmd/lingo -run TestValidateDoesNotCreateRootsOrLockFiles -count=1` | Passed on macOS; missing roots and lock files are not created by validate. |
| Repository/static checks at PR #28 | `go test -race ./...`, `go vet ./...`, `go build ./...`, `go mod verify`, `./scripts/validate-repository.sh .` | Passed on macOS at PR #28. Linux execution was unverified at that point; later evidence follows. |

The local adapter now uses owner-only roots, directory-handle operations,
no-follow file opens, hard-link rejection, no-replace create/install publication,
atomic one-file update, durable attempt markers, directory locks released on
process exit, bounded reads, and explicit `recovery_required` for interrupted
temporary artifacts or uncertain post-publication sync. Existing unknown
artifacts are preserved. The POC still supports only one portable manifest and
no local record replacement.

### PR #29 verification — 2026-09-18

The [POC verification workflow](../../../.github/workflows/poc-verification.yml)
is limited to reproducible verification/acceptance Evidence for this POC. It is
not a product capability, release pipeline or permanent architecture commitment.
Passing jobs do not settle #19–#21, close #14 or replace human Acceptance.

Base: `main` at `9544e38` after merged PR #28. The first verification commit is
`9b70f04c625a5ef35b8f153ea000f55d304cf271`. The
[GitHub Actions run](https://github.com/rgomids/axiom/actions/runs/35398139103)
executed that commit on Ubuntu 24.04 and macOS 15. Both jobs passed
`go test -race ./...`, `go vet ./...`, `go build ./...`, `go mod verify`,
`./scripts/validate-repository.sh .` and `./scripts/dogfood-poc.sh`.
This is runtime Linux evidence, not merely a cross-build. Reproduce the
same checks locally from the PR commit or rerun the workflow.
The follow-up commit `a074ca0346d9420ed61d202523c52f394e7e3eda` added
ACL and post-publication sync fault tests plus structured hash output. Its
[GitHub Actions run](https://github.com/rgomids/axiom/actions/runs/35398936936)
passed the same checks on Ubuntu 24.04 and macOS 15, including both platform
ACL tests.
The later [run at `21d3d3a`](https://github.com/rgomids/axiom/actions/runs/35399489754)
passed both platforms with the adapter-level injected disk-full, manual
recovery and staged hard-link race tests added. The workflow uses commit-pinned
`actions/checkout` and `actions/setup-go` from `064311f` onward; its
[macOS/Linux run](https://github.com/rgomids/axiom/actions/runs/35399559877)
passed all jobs.
One local macOS run of the revised dogfooding script exited 0: portable
before-install and after-install SHA-256 were both
`8849e6e9ab9eb9f5edf36635ef30b8f71fe8a81d584392c80e7dde50b6994e28`;
after-update SHA-256 was
`6433b90b883b3926383855c9687f330de485a9373f3ae7b522dfd08f11b176c7`.
The UUID is generated per run, so these sample hashes are evidence of equality
and change in that run, not fixed goldens.

| POC obligation / Specification mapping | Reproducible test or command | Observation and limit |
|---|---|---|
| Init, validate, reopen, install, update; AC-01, AC-03, AC-15 subset | `go test ./cmd/lingo -run TestExecutableMinimalLifecycleAndFailurePaths -count=1`; `./scripts/dogfood-poc.sh` | Real executable and bounded Axiom dogfooding run on both platforms. One manifest and name update only. |
| Portable intent vs local state; AC-03, AC-17 subset | Same black-box test; `./scripts/dogfood-poc.sh` | Exact portable bytes unchanged during install; one separate ID-addressed local record; stale local observation after update. |
| Invalid input and conflicts; AC-06, AC-07, AC-13, AC-15 subset | `go test ./cmd/lingo ./internal/manifest ./internal/project ./internal/projectapp` | Parser, identity, CLI and application tests pass. Full Specification matrix remains wider. |
| Cancellation and pre-commit preservation; AC-07, AC-16 subset | `go test ./internal/local -run 'Test(PortableUpdateCancellationBeforePublicationPreservesOldBytes|InstallationCancellationBeforePublicationLeavesNoRecord)' -count=1` | Prior manifest or absent record remains authoritative. |
| Process death before/after publication; AC-07, AC-16 subset | `go test ./internal/local -run 'Test(Portable(Create|Update)CrashBoundaryRequiresRecovery|InstallationCrashBoundaryRequiresRecovery)' -count=1` | Pre-commit leaves no final target or old bytes; post-commit leaves complete new bytes; interrupted artifacts produce `recovery_required`. |
| Post-publication sync failure; AC-07, AC-16 subset | `go test ./internal/local -run 'Test(PortablePostPublicationSyncFailureRequiresRecovery|InstallationPostPublicationSyncFailureRequiresRecovery)' -count=1` | Injected `EIO` after publication leaves complete new bytes and a preserved attempt marker; read/reopen returns `recovery_required`. Physical power-loss durability remains unproved. |
| Concurrent reader/writer and conflicting updates; AC-07, AC-16 subset | `go test ./internal/local -run 'Test(PortableReadersAndWritersConflictWithOtherProcess|PortableConflictingWritersHaveOneWinner)' -count=1` | Process-held exclusive lock rejects reader/writer; two stale updates have one winner. Independent slugs still share a root lock. |
| Symlink, hard link, rename and ancestor replacement; AC-09, AC-16 subset | `go test ./internal/local -run 'Test(PortableStoreRejects|PortableCreateRejectsAncestorReplacementBeforePublication|PortableUpdateRejectsProjectRenameBeforePublication|InstallationRejectsTargetReplacementBeforePublication|PortableRejectsStagedHardLinkBeforePublication|InstallationRejectsStagedHardLinkBeforePublication)' -count=1` | Controlled path replacement and hard-link insertion into staged files fail before publication; outside sentinels unchanged where applicable. Arbitrary hostile same-user interleavings remain unproved. |
| Short write and simulated storage exhaustion; AC-07 subset | `go test ./internal/local -run 'Test(WriteCompleteHandlesShortWritesAndStorageFaults|PortableDiskFullBeforePublicationPreservesPriorState|InstallationDiskFullBeforePublicationLeavesNoRecord)' -count=1` | Partial writes are completed; zero progress and injected `ENOSPC`/`EDQUOT` fail. Adapter-level injected `ENOSPC` after a one-byte temporary write preserves absent/old final state. No physical full-filesystem run. |
| Permission denial; AC-07, AC-09 subset | `go test ./internal/local -run TestPortableUpdatePermissionFailurePreservesOldBytes -count=1` | Unprivileged macOS and Linux runs passed; root execution skips. ACL checks are separate below. |
| Explicit ACL detection; SEC-005, AC-09 subset | `go test ./internal/local -run 'TestPrivate(RootRejectsPermissiveACLDespiteMode0700|FileRejectsPermissiveACL|RootRejectsDefaultACLDespiteMode0700)' -count=1` | macOS extended ACLs and Linux POSIX default ACLs are rejected when mode bits alone appear private; both platform jobs passed. Inherited or concurrent ACL mutation remains a separate race question. |
| Recovery classification; AC-07, AC-16 subset | Crash tests above; `go test ./internal/local -run TestPortableManualRecoveryPreservesEvidenceAndReopens -count=1`; [manual recovery procedure](recovery-poc.md) | Recognized interrupted artifacts fail closed; a controlled operator quarantine preserves the marker and permits reopening old/new complete bytes. No automatic repair or proved recovery after power loss. |

### Remaining gaps for #19–#21

- `#19`: Physical disk-full, injected rename faults and sync/fault stages beyond
  the post-publication case, and exhaustive same-user race proof are incomplete.
  A macOS synthetic directory retained `0700` mode while
  an `everyone` ACL granted list/search; draft PR #29 now rejects such ACLs on
  opened roots/files. The Linux default-ACL test also passed; inherited and
  concurrent ACL changes remain outside the tested interleavings.
- `#19`: `recovery_required` is deterministic for recognized leftovers, with a
  [manual operator procedure](recovery-poc.md). It preserves Evidence and unknown
  artifacts. Automatic recovery is outside this POC; post-publication durability
  uncertainty requires explicit inspection and cannot be reported as a rollback.
- `#20`: The POC subset above has reproducible macOS/Linux Evidence. Full
  AC-01–AC-18 coverage is not claimed: rename, documents, bindings, Runtime,
  guided flows, and Git execution are outside the delivered one-file baseline.
- `#21`: The bounded dogfooding run is recorded and repeatable. Explicit human
  acceptance remains pending. Merge and test success do not mark Specification
  002 or parent #14 Accepted or authorize MVP.
- `gitleaks` is unavailable on the local host. The repository sensitive-file
  checker passed; consolidated scanner coverage remained unverified in that
  historical run. T39 later records a successful gitleaks 8.30.1 scan.

### E2E extension — 2026-09-19

Specification 003 [T39 Evidence](../003-e2e-codex-poc/evidence-t39.md) extends
the bounded lifecycle with installation, Project bindings/resolution, Codex
Runtime skills, GitHub Work Items and the complete workflow. The revised
`scripts/dogfood-poc.sh` runs that path from an unrelated CWD with isolated
roots and bounded fake provider binaries on macOS/Linux. This closes the former
Runtime/Provider/binding test gap for the E2E POC, but does not close #19's
physical power-loss, exhaustive hostile same-UID race or remaining real
filesystem-fault proof gaps. #19 and dependent #20 therefore remain open rather
than treating broader E2E success as filesystem acceptance.

### Parent #14 exit checklist for human review

This assessment is for the bounded one-file POC, not full Specification 002
acceptance. `satisfied` means observed in versioned tests and Evidence;
`partially satisfied` means the POC requirement has a material unproved edge.

| #14 exit item | Assessment | Gap, risk, minimum next work | POC blocker? |
|---|---|---|---|
| Executable local Lingo CLI | Satisfied | Minimal commands exercised as a real binary. | No |
| Init, validate, reopen, explicit update | Satisfied | Only name update and one manifest are supported as the approved POC baseline. | No |
| Portable intent separated from local state | Satisfied | Install keeps exact portable bytes; local record stays under an ID-addressed state root. | No |
| Filesystem safety, atomicity and recovery | Partially satisfied | ACL detection, controlled renames, atomic publication and manual recovery are covered; arbitrary hostile same-user timing and full fault matrix remain unproved. Complete adversarial review or reconcile the threat model. | Yes under SEC-003/005 as currently written |
| Failures never publish partial or invalid state | Partially satisfied | Tested cancellation, crash, injected disk-full and sync-failure stages preserve complete old/new bytes; physical disk-full and remaining rename/fault stages need proof or explicit POC scoping. | Yes until those stages are checked or explicitly scoped out |
| Deterministic tests and reproducible Evidence | Partially satisfied | POC tests and macOS/Linux CI are reproducible; the uncovered #19 matrix prevents complete #20 claim. | Yes, depends on #19 |
| Workflow used on bounded Axiom change | Satisfied | PR #28 dogfooding plus repeatable script in PR #29. | No |
| Limitations and unsupported scenarios documented | Satisfied | One-manifest boundary, manual recovery, missing full Specification features and proof gaps listed above. | No |
| Explicit human acceptance | Not satisfied | Human review and acceptance decision have not occurred. Do not mark Specification 002 or #14 Accepted from a merge or CI run. | Yes, final gate |

Scope limitations already established by #14 and the delivery issues: this POC
does not provide production compatibility, distributed persistence, remote
locking/sync, broad Provider/Runtime integration, Git execution or automated
orchestration. Manual recovery remains the documented POC procedure. Security
and fault-proof gaps above are **not** accepted waivers.

**Human decision required** if the POC is to be accepted without exhaustive
same-user race proof: may SEC-003's adversary be limited to cooperating Lingo
processes plus controlled path replacement checks for this one-file POC, with
malicious concurrent mutation by another process under the same UID explicitly
unsupported? If not, complete the stronger confinement proof before acceptance.
