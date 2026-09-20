# Evidence — T35 Project configuration

## Delivered behavior

- `lingo project configure` supports a direct prompt flow or repeatable flags:
  `--slug`, `--name`, and repeatable `--repository key=/absolute/path`.
- One application path creates a complete domain-validated portable Project,
  then writes exact machine-local repository bindings through the strict local
  record codec/store.
- Portable `axiom.yaml` contains repository keys only; absolute paths occur only
  in protected local state.
- The existing `$axiom-project-configure` skill delegates to this command.
- Equivalent retry is a no-op. Conflicting Project intent or local bindings
  require explicit update/replacement rather than silent mutation.

## Reproduction

```bash
go test ./cmd/lingo -run Configure -count=1
go test ./internal/cli -run 'Interactive|Delegates' -count=1
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

Tests cover guided input, non-interactive black-box configuration, immediate
global resolution, duplicate-key rejection before publication, portable/local
path separation, and truthful `portable_committed_local_failed` reporting when
the local destination becomes unsafe after composition.

## Atomicity boundary

Portable and local publication are distinct commits under H10/H11. All input and
domain errors are rejected before either write. A failure after portable commit
cannot be reported as rollback; Lingo returns `portable_committed_local_failed`
and leaves the valid portable Project inspectable for explicit reconciliation.
