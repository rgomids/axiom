# Quality Policy

## Evidence hierarchy

Prefer:

1. executable tests/checks;
2. static analysis;
3. reproducible commands;
4. concrete diff inspection;
5. reasoned judgment.

## Review expectations

Review proportional to risk:

- behavior against acceptance criteria;
- test adequacy;
- regression risk;
- dependency direction;
- unnecessary complexity;
- concurrency/error handling where relevant;
- observability;
- operational impact;
- documentation consistency.

Avoid universal style dogma such as arbitrary function line limits.

## Change discipline

- Keep changes scoped.
- Avoid opportunistic refactors unless they directly reduce implementation risk.
- Preserve existing behavior unless change is intentional.
- When modifying legacy areas, prefer minimal, reversible changes.

## Persisted-state compatibility

A newer release must never refuse persisted state that an earlier release
legitimately wrote. Three executable guards enforce it; do not weaken them:

- `TestEveryStateRootWriterDirectoryIsInventoried` (`internal/local`): every
  state-root directory a store opens is classified by the inventory
  (`StateRootDirectories`).
- `TestEveryV1WriterIsRecognizedByInventory` (`internal/local`): every v1 kind
  (`V1Kinds`) is produced by a real store write and read back as supported.
- `TestStableCorpusResolvesToDirectUpgrade` and `TestStableCorpusIsAppendOnly`
  (`internal/compatibility`): the frozen corpus in
  `internal/compatibility/testdata/stable-v1` keeps resolving to a `direct`
  upgrade and is never edited or pruned.

When a change adds a persisted file, directory, or kind, extend the inventory,
exercise the writer in `writeEveryV1Kind`, and append it to the corpus with
`AXIOM_FREEZE_STATE_CORPUS=1 go test ./internal/local -run
TestEveryV1WriterIsRecognizedByInventory` before release. A new persisted
format follows FR-026 instead: declare its compatibility window and forward
path, and add its own frozen corpus.
