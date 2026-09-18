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
- `gitleaks` is not installed. `scripts/test-check-projectapp.sh` requires a
  local Go 1.26 toolchain; this host's forced-local toolchain is Go 1.24.5.

## Hardening and dogfooding — 2026-09-18

The bounded Axiom change exercised here is POC persistence hardening on
`agent/finish-poc` at implementation commit
`a704a6367485d3a800a36ee31521d9dd245b5b95`. Run
`./scripts/dogfood-poc.sh` from the repository root.
It creates a temporary Project for that change, validates, reopens, installs,
updates its name, verifies portable bytes stay unchanged during install, verifies
one ID-addressed local record, and observes stale local state after update. Its
temporary files are removed after the run. The local macOS run exited 0 and
emitted the expected JSON categories for all eight commands.

| POC criterion | Executable evidence | Current result |
|---|---|---|
| Minimal lifecycle and failures | `go test ./cmd/lingo -run TestExecutableMinimalLifecycleAndFailurePaths -count=1` | Passed on macOS; real executable covers init, validate, reopen, install, update, no-op, conflict, missing input and stale state. |
| Portable/local separation | Black-box test and `./scripts/dogfood-poc.sh` | Passed on macOS; install preserves exact portable bytes and publishes one separate local record. |
| Create/update confinement | `go test ./internal/local -run TestPortableStore -count=1` | Passed on macOS for symlink ancestor, symlink Project, hard-linked manifest, unknown artifacts and interrupted temporary artifacts. |
| Process coordination | `go test ./internal/local -run TestPortableLockSerializesProcessesAndSurvivesCrash -count=1` | Passed on macOS; conflicting process cannot create while lock held and can proceed after owner death. |
| Permission denial before update | `go test ./internal/local -run TestPortableUpdatePermissionFailurePreservesOldBytes -count=1` | Passed under unprivileged macOS user; prior bytes stay intact. Root execution skips this case. |
| Read-only validation | `go test ./cmd/lingo -run TestValidateDoesNotCreateRootsOrLockFiles -count=1` | Passed on macOS; missing roots and lock files are not created by validate. |
| Repository/static checks | `go test -race ./...`, `go vet ./...`, `go build ./...`, `go mod verify`, `./scripts/validate-repository.sh .` | Passed on macOS. Linux cross-build passed; Linux execution unverified. |

The local adapter now uses owner-only roots, directory-handle operations,
no-follow file opens, hard-link rejection, no-replace create/install publication,
atomic one-file update, directory locks released on process exit, bounded reads,
and explicit `recovery_required` for interrupted temporary artifacts or uncertain
post-publication sync. Existing unknown artifacts are preserved. The POC still
supports only one portable manifest and no local record replacement.

### Remaining acceptance blockers

- Linux execution and filesystem fault evidence are unavailable on this host.
- Disk-full, short-write, broader permission/ACL, crash at each commit boundary,
  concurrent reader, and hostile same-user ancestor replacement matrices are not
  complete. ACL behavior is unverified. Cross-build does not establish runtime
  behavior.
- Interrupted attempt artifacts fail closed with `recovery_required`; there is
  no automated recovery command or proved ownership-based cleanup protocol.
- Specification 002's full T05–T21 and AC-01–AC-18 matrix remain broader than
  this one-file POC. Rename, documents, bindings, Runtime and guided flows are
  unavailable.
- `gitleaks` is unavailable. The repository sensitive-file checker passed, but
  scanner coverage remains unverified.

These gaps keep POC issues #19 and #20 open. #21 dogfooding is recorded, but
explicit human acceptance is still pending. Neither a technical merge nor this
Evidence authorizes MVP work.
