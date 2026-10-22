# AXM-37 — Versioned strict Go verifier amendment r2

Attempt `v1-go-versioned-helper-amendment-r2`. Same Draft
[PR #335](https://github.com/rgomids/axiom/pull/335), branch
`docs/axm-37-offline-operational-documentation`. Preflight local/remote HEAD
`3139cd03b29af78f1fb221e3ddfcf55d54515495`, clean tree. Linear ledger showed r2
dispatch.accepted and the prior attempt terminal blocked, no other active
attempt. The accepted dispatch starts this continuation. No subagents: strictly
lower capability was not proven. Shared agent-context endpoint was advertised
but returned tool-not-found; the card and full comment ledger were available.

## DAG and scope

Preflight → distribution contract → helper and both bootstrap surfaces →
closed producer/verifier/publication inventories → differential/refusal tests
→ exact final-SHA CI and independent review. The normative
[distribution amendment](distribution-helper-v1.md) defines the trusted handoff,
seven assets, budgets, source/host binding and historical compatibility. No
archive member or metadata schema changed. No default-v2 promotion.

## Technical decisions

- TD1: separately built, non-installed stdlib executable, version/full source
  commit/OS/architecture in asset name. Avoid candidate execution, host compiler,
  PATH discovery and custom DEFLATE parsing. Existing shell/Go validators retain
  manifest, identity, permission, lock, quota and publication responsibilities.
- TD2: authenticate SHA256SUMS against GitHub release API `digest` over HTTPS,
  validate closed assets and tag commit, then authenticate helper bytes before
  first `identity` invocation. This preserves existing repository/HTTPS trust;
  it does not claim detached signing or protection from repository compromise.
  Missing API digest refuses. Remote `jq` prerequisite is explicit; no automatic
  tool installation. Offline pin must arrive through an independently trusted
  handoff, never be inferred from an adjacent checksum.
- TD3: Go gzip member loop uses `Multistream(false)` and a byte reader so every
  compressed member and trailing byte is consumed or refused. A single tar spans
  v1 members; empty prefix/suffix members remain historical valid controls. Empty
  archive, corrupt/split-invalid streams and non-gzip tails refuse. v2 remains
  single-member. Stdlib tar validates headers, with explicit two-block zero EOF.
- TD4: stage authenticated helper/archive copies privately, check host binary
  header before helper execution, exclusive tar output only after verification,
  no helper persistence or automatic rollback. Preserve existing installer
  atomicity/recovery and bounded same-UID threat model.
- TD5: historical releases without helper assets need an explicitly pinned
  compatible newer helper; no silent cross-release network lookup. Archived old
  installer bytes are not claimed repaired. Both local flags and explicitly
  supplied environment pins are supported operator inputs.

GitHub evidence for each decision is this PR's final committed amendment,
`cmd/axiom-archive-verify`, `scripts/install.sh`, `install-release.sh`,
`build-release-archives.sh`, `verify-release-artifacts.sh` and regression tests.
Final SHA and immutable URLs are reported in the completion callback rather than
introducing a self-referential commit hash into this file.

## Verification status

Initial implementation tests: macOS 27.0.1 arm64, Go 1.26.1, Apple gzip 487.0.1.
The existing 34-case shell/Go integrity matrix passed after helper integration;
its GNU gzip 1.12 comparator used local `gcc:14-bookworm` with network disabled
and identical fixture paths. The expanded matrix adds compress-format suffix and
split-invalid cases. Helper unit fixtures cover member sequence, CRC, truncation,
trailing zeros/garbage, empty archive and missing tar EOF, with no output on refusal.
Authentication tests cover missing helper, incorrect pin, platform header mismatch
and symlink leaf, with candidate sentinel and absent destination assertions.
The local docs journey passed; it remains PARTIAL qualification, not network/source
isolation or native Runtime evidence. Final exact-SHA reruns and terminal remote
CI results belong to the callback and supersede no historical failure.

Baseline CI at `3139cd03b29af78f1fb221e3ddfcf55d54515495` FAIL on Linux zero_trailing
remains historical FAIL. Earlier executiongraph instability remains separate.
Native Linux amd64/arm64 hardware qualification, Q6 Codex/Claude sessions,
AC-08 and v0.1.0 migration floor: NOT_RUN/UNVERIFIED. Container GNU comparison
is not native Linux qualification. No credentials, paid hosts, real sessions,
release, acceptance, merge, status change or successor unblock.

## Reproduction

```sh
go test ./cmd/axiom-archive-verify ./internal/install -run 'Test(StrictMembersAndNoOutputOnRefusal|ShellArchiveIntegrityBeforeExecution|ShellHelperAuthenticationBeforeExecution|ShellHelperNeedsNoPython)$' -count=1
AXIOM_TEST_GNU_GZIP_CONTAINER=gcc:14-bookworm go test ./internal/install -run '^TestShellArchiveIntegrityBeforeExecution$' -count=1 -v
./scripts/test-install-posix-facade.sh
./scripts/test-install-bootstrap.sh
./scripts/test-release-pipeline.sh
./scripts/test-docs-offline.sh
./scripts/validate-repository.sh .
```

Pre-final source checkpoint `a7276c8df1f999afb40ada954c4a919479e5b36e`:
release pipeline PASS (three helpers/archives, both generations, cross-build
linkage and no-publication contract); focused expanded 38-case shell/Go matrix
with Apple/GNU comparators PASS; helper unit and five authentication cases PASS;
no-Python controls PASS; POSIX facade 15 cases PASS; `go vet` on affected packages
and module tidiness PASS. Bootstrap exercised install/reinstall/upgrade, races,
recovery and authenticated-metadata refusals; one version-substitution fixture
corrupted checksum text and was corrected to replace only literal asset prefixes.
Final rerun must pass before completion. Local Staticcheck binary was unavailable;
the exact-SHA CI installs its pinned version. Repository validator PASS at the
initial implementation checkpoint; final rerun belongs to the callback.

Commands above define the reproducible suite, not a blanket PASS assertion. Exact executed outcomes, artifact URLs and remaining work are
reported in the final callback. Independent review is the next gate only when
implementation and exact-SHA checks pass; no human acceptance is inferred.
