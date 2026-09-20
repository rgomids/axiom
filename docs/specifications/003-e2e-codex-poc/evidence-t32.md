# Evidence — T32 local installation

## Delivered behavior

- `scripts/install-axiom.sh` builds Lingo from the checked-out Axiom revision.
- Default publication is `$HOME/.local/bin/lingo`; absolute isolated overrides
  support deterministic tests.
- The installer detects whether its destination is an exact `PATH` entry. When
  absent, it returns `pathConfigured: false` plus a shell-safe export instruction;
  it never mutates a shell profile.
- A protected checksum receipt distinguishes an owned installation from unrelated
  or modified content. Idempotent reruns do not rewrite identical bytes.
- `lingo version` reports product, binary, build revision, public source URL and
  whether dirty-source status was known/true. Dirty installs use a `-dirty`
  version suffix and record `dirty=true` in the receipt.
- No local checkout path or credential is embedded in the binary output/receipt.

## Reproduction

```bash
go test ./cmd/lingo ./internal/cli
./scripts/test-install-axiom.sh
bash -n scripts/install-axiom.sh scripts/test-install-axiom.sh
go test ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

The installer test starts with its destination absent from `PATH`, verifies the
returned safe export guidance, invokes the exact installed `lingo version` from
`/`, reruns unchanged, tampers with the binary, and proves the next run fails
without overwriting it. A temporary untracked source sentinel proves dirty build
metadata in installer output, `lingo version` and the receipt.

## Limitations

This is a source-checkout POC installer, not a package manager, updater, signer or
notarized release. PATH configuration is instructed but never persisted or mutated.
