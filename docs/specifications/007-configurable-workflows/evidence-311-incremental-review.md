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

## Windows support removal — Block 1/5 (Go code), 2026-10-13

Sections above are preserved as written at their time; this section supersedes
their open CR-016 items without rewriting them.

Observed after their publication: commit
`707765d2b7d5d0dceb370a01e830e106b6e779ed` ran in
[CI run 38451927783](https://github.com/rgomids/axiom/actions/runs/38451927783).
The [native Windows job 115485169528](https://github.com/rgomids/axiom/actions/runs/38451927783/job/115485169528)
exited 1 on the five invocation/identity tests (invalid adapter configuration);
Linux and macOS jobs passed and aggregate verify failed on Windows. Rafael then
approved a timeboxed Option A (native Windows pinned-Axiom invocation). A Linux
investigation found no verifiable safe Windows `Bash(...)` permission
representation; that attempt was never committed.

**Human decision:** [PR #311 comment 6153404308](https://github.com/rgomids/axiom/pull/311#issuecomment-6153404308)
removes Windows support entirely. Supported platforms are Linux and macOS.
CR-016, Option A and fallback B are **CANCELLED_BY_HUMAN_DECISION**. The
decision authorizes no `t.Skip`, merge, release, new ADR or Windows support
under another form; PR #311 stays the single PR.

Block 1 scope (production Go, related Go tests, this Evidence only):

- Removed 40 files: every `*_windows.go`/`*_windows_test.go` under `cmd/` and
  `internal/`, the `internal/windowsfs` package, `cmd/lingo/install_windows_result*.go`,
  `internal/cli/windows_permissions.go` (`windows-permissions restore`), and
  test shims that existed only to branch for Windows
  (`cmd/lingo/platform_test.go`, `internal/{runtimeadapter,graphapplication}/posix_paths_test.go`,
  `internal/testfs/permissions_posix.go`).
- Shared code: `stateRoot` no longer has a Windows branch and `NativeStateRoot`
  refuses `windows` (`ErrUnsafe`); `WindowsStateRoot`, the `windows/amd64` host
  row, the Windows managed-settings path, the `USERPROFILE` home fallback and the
  Windows executable-mode bypass were removed. Runtime executable identity now
  requires the exact basename (`codex`/`claude`) instead of tolerating `.exe`, a
  narrowing. `!windows` build constraints became `unix`; fallback files dropped
  `!windows`. A `windows/amd64` build now fails to compile; no Windows stub was added.
- Tests: Windows-only tests and `GOOS == "windows"` skips/branches were removed;
  POSIX assertions were unwrapped, not weakened. `testfs` helpers that were
  identity/no-op on POSIX were inlined. Tests named for Windows state wiring
  that also run on POSIX upgrades were renamed `...WithPersistedSkillsWiring`.
- Preserved: lexical refusal of Windows path spellings and of `cmd`/`powershell`/`pwsh`
  shells (planner, compiler, gitworkspace, scheduler, validator policy),
  `validatorpolicy.portableAbsolute` (persisted-policy structure), the
  `windows-credential-manager` test vectors, frozen-corpus Windows JSON-path
  normalization, goldens and snapshots. No dispatch, Runtime or integration
  authority was broadened.

Validation (Linux host, Go 1.26 toolchain):

- `gofmt -l cmd internal`: no output. `go vet ./...`: PASS.
  `GOOS=darwin GOARCH=arm64 go vet ./...`: PASS.
- `go test` for `./internal/runtimeadapter/...`, `./internal/workflowdefinition/...`,
  `./cmd/lingo/...`: PASS. No `unsafe_source` occurred.
- `./internal/executiongraph/...`: FAIL under the inherited host environment
  (`TestInheritedSnapshotKeepsExplicitEnvironmentValidation`), also at
  unmodified `707765d`; PASS with a minimal environment. Pre-existing and
  host-environment dependent; not changed here.
- `go test ./...`: FAIL only in `TestStableCorpusIsAppendOnly`, caused by the
  pre-existing untracked snapshots `v0.12.0-issue276.{b5t34recovery,bld007}`
  (CR-015 freeze gate); a clean worktree of `707765d` plus this diff: PASS.
- `go build ./cmd/lingo` with `CGO_ENABLED=0` for linux/amd64, linux/arm64,
  darwin/arm64, darwin/amd64: PASS. No darwin test binary was executed;
  native macOS acceptance remains with CI.

Not yet done (later blocks, same branch): installation/release scripts and
PowerShell entry points (Block 2), CI workflows and rulesets (Block 3), site,
README, specifications and changelog (Block 4), repository-wide sweep (Block 5).
Until Block 2/3, release scripts that build `windows/amd64` and the Windows CI
jobs are expected to fail. Removal is **not complete** before Block 5.
No merge, release, tag, ruleset change or review-thread closure was performed.
This section is committed with the Block 1 change, whose parent is `707765d`.

## Windows support removal — Block 2/5 (distribution scripts), 2026-10-13

Base `00fa0d7b1505fc00a97d9bced0d096063dfebcae` (Block 1). Block 1 is not rewritten.

Scope (scripts and the SA9003 cleanup only; no workflow, ruleset, doc or frozen
corpus change):

- SA9003: the empty `if` branches left by Block 1 in
  `internal/codexruntime/retirement_test.go` and
  `internal/install/poc_transition_test.go` only called a Windows skip helper;
  they were removed. Every case and assertion is kept.
- `build-release-archives.sh` builds only `macos-27/arm64`, `linux/amd64` and
  `linux/arm64`; no `axiom.exe`, `install.ps1` or `install-release.ps1` is
  bundled. Names, `MANIFEST.sha256`, metadata and `SHA256SUMS` of the POSIX
  bundles are unchanged.
- `verify-release-artifacts.sh` requires exactly the three POSIX archives; a
  `windows-amd64` archive is outside the closed set and refused.
  `install-release.sh` already accepted only POSIX rows (unchanged).
- Deleted `scripts/{install,install-release,test-windows-install,test-windows-bootstrap}.ps1`
  and only their four `automation-registry.json` entries (required for the
  registry check to stay valid).
- `smoke-prepared-artifact.py`, `accept-native-artifact.py` accept only
  supported rows; Windows ACL/PowerShell/environment plumbing removed; POSIX
  archive-entry, checksum, permission, provenance and upgrade checks unchanged.
  `acceptance/stage-plan-r1.py` dropped its `os.name == "nt"` branches.
- Release tests expect three archives (four assets with `SHA256SUMS`);
  `test-release-pipeline.sh` adds an unsupported-row refusal case and
  `test-native-acceptance.py` replaces the Windows alias test with a refusal
  of unsupported rows.

Validation (Linux amd64, Go 1.26.0, Staticcheck 0.8.1):

- `check-go-quality.sh all`, `test-go-quality.py`, `go vet ./...`,
  `GOOS=darwin GOARCH=arm64 go vet ./...`, `check-automation-registry.py`,
  `test-check-automation-registry.py`: PASS.
- `go test ./...` in a clean worktree with a minimal environment: PASS. In the
  working checkout the known `TestStableCorpusIsAppendOnly` (untracked CR-015
  snapshots) and inherited-environment failures recur, as in Block 1.
- `test-release-pipeline.sh`, `test-release-flow.sh`, `test-delivery.sh`,
  `test-release-corrections.py`, `test-release-pr-checks.py`,
  `test-code-health.py`, `test-install-bootstrap.sh`,
  `test-install-posix-facade.sh`, `test-native-acceptance.py`,
  `test-prepared-artifact-smoke.py`: PASS.
- `test-upgrade-journeys.sh` with a candidate built at this commit, published
  `v0.16.0` and `v0.1.1` `linux-amd64` assets and the `v0.1.0-poc.1` binary: PASS.
- Clean-clone build plus `verify-release-artifacts.sh`: PASS for exactly three
  archives; `sha256sum -c SHA256SUMS`: PASS; no `.exe`/`.ps1` entry in any
  archive; adding a `windows-amd64` archive: refused.
- Pre-existing, not Block 2: `test-release-archives.sh` (not run in CI)
  still asserts the former two-skill surface (lines 65, 111, 125, 150-170)
  against today's four skills; its archive, checksum and clean-clone provenance
  sections pass. `acceptance/test_stage_plan_r1.py` has the same 67 failures
  at `00fa0d7`.
- macOS: darwin/arm64 is cross-built and verified by header and build
  information only; no darwin binary was executed (CI).

Known until Block 3: `.github/workflows/ci.yml` Windows verify steps call the
deleted PowerShell scripts, and `native-artifact-acceptance.yml` /
`native-acceptance-qualification.yml` pass `--row windows-amd64`, which the
scripts now refuse; Windows CI jobs are expected to fail. Removal is **not
complete** before Block 5. No merge, release, tag or ruleset change was made.
