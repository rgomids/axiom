# PR #311 incremental review corrections — CR-015 / CR-016

Scope: test fixtures and investigation only, based on reviewed HEAD
`11f88c1d285ffbedf1c32cf20eadabd040cb3f28`. No production builder, identity
encoding, capability boundary or CR-014 fail-closed composition changed.
No skips were added and no accepted golden values changed.

## CR-016: native path fixtures and remaining contract discrepancy

Native Windows baseline evidence: [CI run 38444379041, Windows job
115458901294](https://github.com/rgomids/axiom/actions/runs/38444379041/job/115458901294),
`windows-2022`, `go test ./... -timeout 10m`, reviewed HEAD above.

| Failing test | Root cause at reviewed HEAD |
| --- | --- |
| `TestAdapterVersionCanonical` | `identityRequest` uses `/bin/codex`, `/work`, `/inbox`, `/bin/axiom`; native `filepath.IsAbs` rejects volume-less Windows paths, so `ModeDigest` cannot produce a binding. |
| `TestModeDigestGolden` | Same invalid request fails in `BuildStageInvocation` before identity projection; the observed digest is empty, not a changed golden vector. |
| `TestModeDigestVolatileNormalization` | Both base and alternate fixtures use POSIX paths; both return invalid adapter configuration before normalization. |
| `TestModeDigestSafetyDifferences` | Same builder rejection; ignored errors make two empty digests appear to collapse confinement variants. Errors are now asserted. |
| `TestTaskPacketAcceptsBenignPrefixText` | `packetFixture` uses `/state/inbox/attempt-1` and `/bin/axiom`; `BuildTaskPacket` rejects them through `filepath.IsAbs` before considering benign text. |
| `TestStageInvocationCorrelationIsAttemptOwned` | Its literal POSIX request fails path validation before attempt-owned environment construction. |

Fixtures now preserve those layouts using the native temporary-directory volume,
native root and `filepath.Join`. Alternate identity paths are also native; the
normalization test searches for JSON-escaped path values so Windows backslashes
cannot mask a leak. Added tests exercise accepted canonical native stage paths,
root separation/containment, noncanonical and relative stage-path rejection,
and accepted native packet paths with relative/NUL/chat-marker refusals.

**Remaining blocker:** `BuildStageInvocation` separately rejects `:` and `\`
in `AxiomPath` for its restricted `Bash(...)` tool pattern, for both adapters.
A canonical drive path contains both; a canonical UNC path contains backslashes.
Thus the five invocation/identity positive tests still cannot succeed on native
Windows under the current contract. Correcting only POSIX fixtures is insufficient.
The v1 identity projection normalizes path roles and the pinned executable; no
golden discrepancy has been demonstrated. All golden vectors remain unchanged.

CR-016 remains **OPEN, partially corrected**. Completion requires an explicitly
approved Windows pinned-Axiom/tool-pattern contract that preserves security, then
native Windows CI evidence for the exact corrected commit. This change grants
no such production-contract authority. The six positive tests remain enabled.
The new commit is local only; no native Windows execution of its bytes is claimed.

## CR-015: freeze boundary, inputs and acceptance dependency

`InventoryDispatchInputs` occurs exactly once in `additiveKinds` in
`internal/local/inventory.go`. The stable-corpus test continues to require every
kind from both `V1Kinds()` and `AdditiveKinds()`. No inventory or guard edit is
needed. CR-015 remains **OPEN**, pending separate freeze/release authority.

After that authority is granted, use an isolated clean checkout of the exact
authorized writer revision, with the authorized release label matching
`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`. Run:

```sh
AXIOM_FREEZE_STATE_CORPUS="$AUTHORIZED_RELEASE_LABEL" go test ./internal/local -run '^TestEveryV1WriterIsRecognizedByInventory$' -count=1 -v
AXIOM_FREEZE_STATE_CORPUS= go test ./internal/local ./internal/compatibility -count=1
```

Inputs are that revision's real `writeEveryV1Kind` store output, including
`CommitDispatchInputs`, its temporary `projects` and `state` trees, the existing
stable-v1 MANIFEST/snapshots, and the authorized release label. `freezeSnapshot`
normalizes the machine root to `/axiom-corpus`, including JSON-escaped Windows
paths, then compares SHA-256 maps of logical paths against complete snapshots.
Identical output adds nothing; distinct output creates a new complete snapshot
and appends MANIFEST entries. Existing labels with different bytes are refused.
Never edit historical snapshots or MANIFEST entries manually.

Review only generated additions, record revision/label/scope in PROVENANCE, and
validate append-only and direct-upgrade guards with freeze disabled before release
acceptance. Follow the inventory's first-stable-release declaration/promotion rule
under that separate release scope. Passing CI or a generated candidate alone is
not acceptance or publication authority. No freeze was executed in this task.

## Validation and limits

Linux, Go 1.26.0:

- `go test ./internal/runtimeadapter/...`: PASS, including unchanged goldens.
- `go test ./cmd/lingo/...`: PASS.
- `go vet ./...`: PASS.
- `go test ./internal/compatibility/...`: PASS in the supplied dirty workspace;
  **FAIL** in an isolated `git archive` of reviewed HEAD, solely because
  `stage_dispatch_inputs` is absent from the frozen corpus.

The workspace already contained an altered MANIFEST and untracked snapshots
`v0.12.0-issue276.b5t34recovery` and `v0.12.0-issue276.bld007`, plus
`.claude/worktrees/`. They were neither changed nor included in this correction.
Their effect on compatibility does not satisfy the freeze gate.

Windows: the linked native baseline confirms all six failures and the separate
stable-corpus failure. Corrected-commit native CI remains **NOT_RUN**; cross
compilation is not substituted. macOS corrected-commit execution is **NOT_RUN**.
The existing CI Windows job already runs the native suite; it needs access to
the corrected revision after separately authorized publication of that branch.
No push, merge, release, review-thread closure, real Runtime or probe was executed.
