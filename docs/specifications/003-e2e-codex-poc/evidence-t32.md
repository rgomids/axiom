# Evidence — T32 local installation

## Delivered behavior

- `scripts/install-axiom.sh` builds Lingo from the checked-out Axiom revision.
- Default publication is `$HOME/.local/bin/lingo`; absolute isolated overrides
  support deterministic tests.
- A protected checksum receipt distinguishes an owned installation from unrelated
  or modified content. Idempotent reruns do not rewrite identical bytes.
- `lingo version` reports product, binary, build revision and public source URL.
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

The installer test uses isolated absolute roots, invokes `lingo version` from
`/`, reruns unchanged, tampers with the installed binary, and proves the next run
fails without overwriting that content.

## Limitations

This is a source-checkout POC installer, not a package manager, updater, signer or
notarized release. Default PATH configuration is documented but not mutated.
