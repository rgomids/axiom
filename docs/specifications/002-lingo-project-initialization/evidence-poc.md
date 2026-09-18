# POC — Lingo minimal Project lifecycle Evidence

## Delivered baseline

The POC integration branch provides a local executable path:

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
- The root-local lock is a POC coordination mechanism. Crash recovery, hostile
  ancestor replacement, hard-link attacks, and multiprocess proof are unverified.
- Local bindings, credentials, Runtime observation, rename, Git, remote sync,
  provider integration and orchestration are unsupported.
- `gitleaks` is not installed. `scripts/test-check-projectapp.sh` requires a
  local Go 1.26 toolchain; this host's forced-local toolchain is Go 1.24.5.

Human acceptance of this POC and any subsequent delivery scope remain required.
