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

## Windows support removal — Block 5/5 (final reconciliation and audit), 2026-10-13

Base `b57e54f30a15201308cd237586400431057c377a` (Block 4). Blocks 3 and 4 are
recorded by commits `d9264d0` (CI and automation contracts) and `b57e54f`
(product and site docs); earlier sections are not rewritten.

**Human decision (final):** [PR #311 comment 6153404308](https://github.com/rgomids/axiom/pull/311#issuecomment-6153404308)
plus the Block 5 instruction from the maintainer: no Windows mention remains in
user-facing content; the only public platform statement is
"Supported hosts: macOS arm64 and Linux amd64/arm64". Allowed remaining
references are exactly H1 (historical Specifications/Evidence, including
Superseded Specification 005), H2 (ADR history), H3 (historical changelog),
H4 (frozen snapshots/testdata) and E1 (negative regression assertions).

### Changes

- Public: `README.md`, `docs/README.pt-BR.md` and `site/index.html` drop the
  "not currently supported"/"may be reconsidered" text (and its Portuguese
  translation) and state the approved sentence verbatim (the pt-BR README and
  the site's `data-pt` keep the English sentence). No replacement negative notice.
- `scripts/validate-landing-page.sh` requires the approved sentence, no longer
  requires the old notice and adds a case-insensitive `windows` refusal for
  `site/`; the PowerShell/installer/`installation.md#windows`/NTFS refusals stay.
- Operational docs: `docs/installation.md` and `docs/commands.md` lose the
  Windows table row, prerequisites, PowerShell bootstrap, PATH, version,
  custom-directory, isolated-test, onboarding/repair and offline sections, and
  the `axiom.exe`/`install.ps1` bundle members; the archive-row sentence is the
  approved one. `docs/development/getting-started.md` drops native Windows test
  storage and the `python.exe` hint; `prepared-artifact-smoke.md` and
  `docs/agent-harness.md` drop Windows claims. `native-platform-acceptance.md`
  now opens with the current POSIX contract (Linux ARM64 lifecycle; Linux AMD64
  and macOS ARM64 smoke), removes the Windows client row and Windows follow-up
  from the operational sections, and moves the Server amendment, authority and
  feasibility sections verbatim under `## Historical Evidence` with a withdrawal note.
- ADR-0010 (Windows filesystem boundary), ADR-0015, ADR-0019 and ADR-0021 get a
  dated withdrawal note at the end of `## Status` naming the affected contract,
  the decision link and what stays valid; the index entries link it. ADR-0018
  admits only `> Superseded by [ADR-…]` annotations that name a superseding ADR,
  and no ADR may be created, so these are plain historical notes, not
  supersession annotations; statuses stay `Accepted`. No text was removed.
- CR-020: `scripts/automation-registry.json` lines 860, 882 and 3091 now say
  three archives (macOS arm64, Linux amd64, Linux arm64), matching
  `build-release-archives.sh` and `verify-release-artifacts.sh`. **RESOLVED.**
- Registry checker: `powershell` is no longer an accepted `runtime`; `.ps1`/`.psm1`
  stay governed extensions so a reintroduced script is detected. Two regression
  tests (refused runtime, unregistered `.ps1`) were added; the runtime test fails
  if `powershell` is restored (mutation checked).
  `.agents/policies/repository-automation.md` matches.
- Issue labels: policy, tests and the three forms were already POSIX-only after
  Block 3; no change. No remote label or ruleset was touched.
- `scripts/periodic-monitor.py`: the `.exe` mention described the upstream npm
  wrapper's inert placeholder, not Windows support; only the comment wording
  changed. The ELF check and disabled lifecycle scripts are unchanged.
- `.gitattributes`: the comment no longer names Windows and the unused
  `*.exe binary` rule is removed (no tracked `.exe`). `scripts/test-release-flow.sh`
  fixture commit subject renamed to "native platform support (#149)"; the
  assertion is unchanged.
- `docs/research/ci-cd-quality-strategy.md` (dated research, 2026-10-05/06) gets
  a historical note; its body is preserved.

### Audit

Commands (from the repository root; `.claude/worktrees/**` excluded because it
holds another session's untracked worktree, not repository content):

```text
rg -n -i 'windows' . --hidden -g '!.git/**' -g '!.claude/worktrees/**'
rg -n -i 'GOOS|GOARCH|win32|powershell|pwsh|\.ps1|\.exe|windows-amd64|windows/amd64|platform:windows' . --hidden -g '!.git/**' -g '!.claude/worktrees/**'
rg --files --hidden -g '!.git/**' -g '!.claude/worktrees/**' | rg -i 'windows|win32|powershell|pwsh|\.ps1|\.exe'
```

- Before (clean worktree at `b57e54f`): primary 610 lines in 100 files;
  secondary 1124 lines; filenames 9. The Block 5 brief's baseline of 639 lines
  in 76 files used a different scope and was not reproduced.
- After (this commit, this Evidence file included): see the totals at the
  end of the list. Secondary 1102 lines; almost all are
  `GOOS`/`GOARCH`/`goos` build variables or `.exe` substrings of identifiers
  (`exec_module`, `ExecutionID`, `execCommand`), not Windows references. The
  Windows-specific secondary lines are classified below. Filenames: 9, all H1/H2
  (`docs/specifications/005-windows-native-support/*` and ADR-0010/0019/0021).
- Public content: zero `windows` matches in `README.md`, `docs/README.pt-BR.md`,
  `site/`, `docs/installation.md`, `docs/commands.md`, `CONTRIBUTING.md`,
  `SUPPORT.md`, `docs/security/`, `docs/product/` and `.agents/`.

Classification justification:

- **H1:** dated Specifications, Evidence, plans and tasks under
  `docs/specifications/`, `docs/development/evidence-*.md`, the sanitized run
  report `native-platform-acceptance-current-evidence.json`, the
  `## Historical Evidence` part of `native-platform-acceptance.md`, the dated
  validation section of `change-aware-ci.md` and the dated research record
  `docs/research/ci-cd-quality-strategy.md` (classified H1 as a dated research
  snapshot; flagged for reviewer confirmation).
- **H2:** ADR files and the ADR index; `scripts/test-check-adr-governance.py`
  names ADR-0010 by its immutable file path as a governance fixture. ADR-0026:61
  ("crash windows") uses the word for time intervals.
- **H3:** `CHANGELOG.md` and `.github/delivery-corrections.txt` (append-only
  correction ledger whose existing rows must not change under ADR-0010 pinned corrections).
- **H4:** the rendered golden `structure-spoof.golden.md`, frozen-corpus
  normalization tests and persisted-state credential-kind vectors
  (`windows-credential-manager` as an opaque historical source kind).
- **E1:** refusals of Windows path spellings and `cmd`/`powershell`/`pwsh`
  shells (planner, compiler, locator, validation, gitworkspace, scheduler,
  validatorpolicy), `NativeStateRoot` refusal test, unsupported-platform and
  unsupported-row tests, label-policy and label-inference negatives, CI-workflow
  absence checks and the landing-page/registry validators.
  `internal/validatorpolicy/policy.go:127` documents that persisted policy
  structure recognizes drive/UNC spellings, but host-native validation refuses
  them before execution (Block 1 preserved it); flagged for reviewer confirmation.
- **Not an OS reference:** `internal/local/dispatch_authority_test.go:34`
  (`TestDispatchCompanionFaultWindows`, fault time windows).
- **R1 = 0, U1 = 0.**

Windows-specific secondary lines outside the primary list: E1 —
`.agents/policies/repository-automation.md:74,75,101`, `scripts/check-automation-registry.py:48`,
`scripts/test-check-automation-registry.py:195-202`, `scripts/validate-landing-page.sh:204`,
`internal/gitworkspace/validator.go:90,91`, `internal/executiongraph/scheduler.go:337,338`,
`internal/validatorpolicy/policy_test.go:75`, `internal/validatorpolicy/portable_test.go:6`,
`internal/workflowdefinition/definition_test.go:88`; H1 —
`docs/development/native-platform-acceptance.md:107,165,171,181`,
`docs/research/ci-cd-quality-strategy.md:557`,
`docs/specifications/004-mvp-v1-baseline/evidence-issue-189.md:14,21,22,41,64,65`,
`docs/specifications/004-mvp-v1-baseline/issue-131-skill-arguments.md:119`,
`docs/specifications/006-installer-host-eligibility/spec.md:61,69` and this file;
H2 — `docs/decisions/0015-installer-host-eligibility-os-family-architecture.md:39,138`.

### Validation

Linux amd64 host, Go 1.26.0 toolchain (`GOTOOLCHAIN=local`), Staticcheck 0.8.1.
Commands needing Git state, network or a clean tree ran outside the sandbox in a
clean worktree of `b57e54f` plus this diff; the working checkout carries
untracked CR-015 snapshots and another session's `.claude/worktrees/`.

- PASS (exit 0): `check-automation-registry.py` (91 surfaces),
  `test-check-automation-registry.py` (33), `test-issue-label-policy.py`,
  `test-periodic-monitoring.py`, `test-ci-classification.py`,
  `check-adr-governance.py` (27 ADRs), `test-check-adr-governance.py` (56),
  `validate-landing-page.sh .`, `test-native-acceptance.py`,
  `test-prepared-artifact-smoke.py`, `test-release-pipeline.sh`,
  `test-release-flow.sh`, `test-delivery.sh`, `test-release-corrections.py`,
  `test-release-pr-checks.py`, `test-code-health.py`,
  `check-go-quality.sh all`, `test-go-quality.py`, `go vet ./...`,
  `GOOS=darwin GOARCH=arm64 go vet ./...`, `go build ./...`, `go mod verify`,
  `go test -count=1 ./...` (minimal environment), `validate-repository.sh .`
  and `test-upgrade-journeys.sh` (candidate built at this commit, published
  `v0.16.0` and `v0.1.1` `linux-amd64` assets, `v0.1.0-poc.1` binary;
  `failures=0`).
- Clean clone at this commit: `build-release-archives.sh`,
  `verify-release-artifacts.sh` and `sha256sum -c SHA256SUMS`: PASS for exactly
  the three archives; no `.exe`/`.ps1` entry in any archive. A build from a
  `git worktree` lacked embedded VCS stamping and failed verification; that is
  a build-location artifact, and the clean clone passes.
- Landing-page mutations, each refused by the intended guard: an added
  "Windows 11" claim, the old notice, an `install.ps1` link, a
  `windows-amd64` download, and removal of the approved sentence.
- Environment-only first attempts, not code failures: `test-ci-classification.py`
  needs a `python` executable (supplied as a link to `python3`);
  `validate-repository.sh` refuses the untracked `.claude/worktrees/` in the
  working checkout; go-quality needs the 1.26.0 toolchain and network.
- Not run locally: `go test -race`, darwin execution and Windows-absence of
  remote checks. They belong to CI on the pushed SHA and are reported in the
  PR acceptance report, not in this commit.

Primary-search exceptions by class (path: lines):

H1:
- `docs/development/change-aware-ci.md`: 86
- `docs/development/evidence-code-health-245.md`: 3, 25, 29, 32, 42, 56
- `docs/development/evidence-go-quality-242.md`: 50, 58, 66, 68, 85
- `docs/development/evidence-issue-133.md`: 19, 34, 36, 38, 39, 40, 47, 49, 52, 53, 56, 58, 74, 86, 88, 104, 113, 114
- `docs/development/evidence-periodic-255.md`: 10, 29, 31, 46, 168, 173, 179
- `docs/development/native-platform-acceptance-current-evidence.json`: 7, 10, 120, 121, 131, 132, 138, 141, 145, 267, 268, 272, 273, 279, 282
- `docs/development/native-platform-acceptance.md`: 88, 89, 96, 97, 98, 100, 111, 127, 130, 131, 138, 143, 163, 164, 169, 173, 174, 177, 217, 228, 229, 233, 245, 301, 303, 312, 313, 319, 326
- `docs/research/ci-cd-quality-strategy.md`: 23, 57, 58, 104, 147, 151, 222, 230, 250, 255, 294, 297, 299, 301, 310, 313, 419, 446, 449, 476, 477, 496, 502, 538, 556, 585, 602, 763, 781, 796, 814, 822, 842, 848, 863, 864, 887, 888
- `docs/specifications/002-lingo-project-initialization/clarifications.md`: 257
- `docs/specifications/002-lingo-project-initialization/evidence-306-implementation.md`: 20, 21, 27, 31, 75, 85, 101, 110
- `docs/specifications/002-lingo-project-initialization/evidence-issue-140.md`: 27
- `docs/specifications/002-lingo-project-initialization/evidence-issue-231.md`: 97, 98, 99, 123, 143, 180, 194, 244, 255, 283, 300, 319, 330, 414, 425
- `docs/specifications/002-lingo-project-initialization/evidence-issue-233.md`: 22, 34, 38, 40, 42, 58, 63
- `docs/specifications/002-lingo-project-initialization/evidence-t03.md`: 294
- `docs/specifications/002-lingo-project-initialization/plan-306-local-runtime-profiles.md`: 342
- `docs/specifications/002-lingo-project-initialization/plan.md`: 207, 452, 511, 513
- `docs/specifications/002-lingo-project-initialization/security-review-306-local-runtime-profiles.md`: 50, 153
- `docs/specifications/002-lingo-project-initialization/spec.md`: 292, 293, 426
- `docs/specifications/002-lingo-project-initialization/tasks-306-local-runtime-profiles.md`: 403
- `docs/specifications/002-lingo-project-initialization/tasks.md`: 300
- `docs/specifications/004-mvp-v1-baseline/evidence-136.md`: 12
- `docs/specifications/004-mvp-v1-baseline/evidence-153.md`: 92, 100, 111, 130, 143, 145, 146, 148, 162, 184, 191, 280, 296
- `docs/specifications/004-mvp-v1-baseline/evidence-186.md`: 61, 99, 110, 111, 115, 117, 122, 125, 131, 132, 133, 141, 142, 143, 145
- `docs/specifications/004-mvp-v1-baseline/evidence-229.md`: 162
- `docs/specifications/004-mvp-v1-baseline/evidence-230.md`: 112, 123, 125, 165, 166, 178, 179, 195
- `docs/specifications/004-mvp-v1-baseline/evidence-issue-137.md`: 128
- `docs/specifications/004-mvp-v1-baseline/evidence-issue-189.md`: 1, 6, 16, 35, 37, 42, 48, 57, 92, 93, 94, 95, 110, 116, 120
- `docs/specifications/004-mvp-v1-baseline/evidence-pr-267-268-integration.md`: 96, 153
- `docs/specifications/004-mvp-v1-baseline/evidence-s7.md`: 228
- `docs/specifications/004-mvp-v1-baseline/evidence-s8.md`: 307
- `docs/specifications/004-mvp-v1-baseline/issue-131-skill-arguments.md`: 68, 74, 75, 87, 90, 99, 102, 105, 106, 112, 115, 117, 171, 182
- `docs/specifications/004-mvp-v1-baseline/issue-272-cli-subscription-auth-preflight.md`: 128
- `docs/specifications/004-mvp-v1-baseline/plan.md`: 193
- `docs/specifications/004-mvp-v1-baseline/spec.md`: 528, 1591
- `docs/specifications/004-mvp-v1-baseline/tasks.md`: 208, 420, 817, 851, 1035, 1181, 1397
- `docs/specifications/004-mvp-v1-baseline/work-item-interview.md`: 90, 105, 106, 107, 108, 110
- `docs/specifications/004-mvp-v1-baseline/workflow-gates.md`: 83, 115
- `docs/specifications/005-windows-native-support/evidence-issue-265.md`: 1, 9, 20, 21, 22, 23, 24, 33, 46
- `docs/specifications/005-windows-native-support/evidence.md`: 1, 8, 14, 18, 25, 44, 51, 55, 66, 68, 69, 73, 76, 81, 82, 90, 92, 97, 105, 113, 130, 134, 135, 137, 138, 144, 146, 147, 153, 159, 163, 166, 172, 180
- `docs/specifications/005-windows-native-support/issue-265-default-onboarding.md`: 1, 7, 13, 43, 69
- `docs/specifications/005-windows-native-support/plan.md`: 1, 10, 14, 17, 20, 21, 22, 31, 37, 45
- `docs/specifications/005-windows-native-support/spec.md`: 1, 3, 6, 21, 22, 24, 31, 38, 44, 46, 53, 57, 58, 61, 66, 69, 74, 75, 79, 84, 87, 94, 98, 103, 108, 118
- `docs/specifications/005-windows-native-support/tasks.md`: 1, 11, 13, 14
- `docs/specifications/006-installer-host-eligibility/spec.md`: 14, 15, 19, 24, 25, 27, 36, 49, 60, 64, 65, 68, 100, 113, 128, 129, 130, 150, 155
- `docs/specifications/007-configurable-workflows/evidence-276-b0-capability-adapter.md`: 25, 61
- `docs/specifications/007-configurable-workflows/evidence-276-b1-t04-t07.md`: 34, 59, 72
- `docs/specifications/007-configurable-workflows/evidence-276-b1-t06-t08.md`: 24, 45, 54, 66, 67, 80
- `docs/specifications/007-configurable-workflows/evidence-276-t09-t27.md`: 12, 28, 34
- `docs/specifications/007-configurable-workflows/evidence.md`: 54, 64, 90
- `docs/specifications/007-configurable-workflows/issue-273-implementation.md`: 109, 124, 127
- `docs/specifications/007-configurable-workflows/issue-274-implementation.md`: 84, 102
- `docs/specifications/007-configurable-workflows/issue-275-implementation.md`: 288
- `docs/specifications/007-configurable-workflows/issue-303-implementation.md`: 135, 136
- `docs/specifications/007-configurable-workflows/plan-tasks-276-stage-agent-execution.md`: 271, 1056, 1125, 1770, 2270, 2273, 2278, 2280
- `docs/specifications/007-configurable-workflows/spec.md`: 526
- `docs/specifications/README.md`: 108, 110, 114, 116, 117, 118, 121

H2:
- `docs/decisions/0010-windows-native-filesystem-boundary.md`: 1, 8, 10, 14, 15, 22, 28, 29, 31, 32, 37, 41, 42, 55, 59, 69, 71, 76, 78, 79, 84, 85
- `docs/decisions/0014-canonical-maintainer-skills-runtime-discovery.md`: 100
- `docs/decisions/0015-installer-host-eligibility-os-family-architecture.md`: 11, 14, 16, 20, 21, 36, 37, 38, 71, 73, 74, 78, 79, 81, 85, 105, 123, 135, 170
- `docs/decisions/0016-candidate-derived-skill-set-receipt.md`: 34
- `docs/decisions/0018-adr-evolution-and-supersession-governance.md`: 11, 30, 100, 101
- `docs/decisions/0019-windows-default-onboarding-and-permission-repair.md`: 1, 6, 12, 14, 18, 19, 33, 34, 57, 82, 92
- `docs/decisions/0021-windows-server-amd64-native-acceptance.md`: 1, 6, 9, 11, 15, 23, 24, 30, 31, 32, 38, 50, 54, 73, 77, 78, 80
- `docs/decisions/0024-model-profile-effort-ownership.md`: 45, 80
- `docs/decisions/0026-successor-stage-dispatch-graph-format.md`: 61
- `docs/decisions/README.md`: 17, 31, 39, 47
- `scripts/test-check-adr-governance.py`: 199, 201, 207

H3:
- `.github/delivery-corrections.txt`: 31
- `CHANGELOG.md`: 58, 96, 156, 180, 623

H4:
- `internal/githubissues/testdata/render/structure-spoof.golden.md`: 49
- `internal/local/state_compatibility_contract_test.go`: 419, 559
- `internal/local/structure_test.go`: 63
- `internal/manifest/review_regression_test.go`: 92

E1:
- `internal/executiongraph/planner.go`: 354
- `internal/githubissues/classification_test.go`: 224
- `internal/githubissues/render_test.go`: 69
- `internal/local/state_root_test.go`: 16
- `internal/local/validation.go`: 97
- `internal/project/locator.go`: 77
- `internal/validatorpolicy/policy.go`: 127
- `internal/validatorpolicy/portable_test.go`: 16
- `internal/workflowcompiler/compiler.go`: 687
- `internal/workflowcompiler/stage_plan_test.go`: 154, 155
- `scripts/test-ci-classification.py`: 157
- `scripts/test-issue-label-policy.py`: 97, 98
- `scripts/test-native-acceptance.py`: 20, 57, 59
- `scripts/test-release-flow.sh`: 1729
- `scripts/test-release-pipeline.sh`: 117, 119
- `scripts/validate-landing-page.sh`: 198, 199, 200, 202, 203, 205, 207, 208, 210

NON_OS:
- `internal/local/dispatch_authority_test.go`: 34

This Evidence file (H1, including the earlier Block 1/2 sections):
- `docs/specifications/007-configurable-workflows/evidence-311-incremental-review.md`: 10, 12, 16, 25, 34, 39, 40, 42, 63, 91, 94, 98, 106, 108, 109, 110, 114, 116, 121, 122, 123, 124, 127, 128, 129, 130, 132, 133, 134, 136, 138, 141, 165, 170, 179, 186, 188, 192, 197, 217, 226, 228, 229, 232, 239, 253, 254, 256, 259, 261, 263, 266, 283, 285, 298, 299, 300, 309, 310, 311, 312, 327, 332, 333, 342, 345, 385, 386, 391, 435, 436, 437, 438, 439, 440, 456, 461, 462

Totals after this commit (primary search):

```text
TOTAL lines=607 files=91
H1: 481 lines / 57 files
H2: 85 lines / 11 files
H3: 6 lines / 2 files
H4: 5 lines / 4 files
E1: 29 lines / 16 files
NON_OS: 1 lines / 1 files
R1: 0 lines / 0 files
U1: 0 lines / 0 files
```
