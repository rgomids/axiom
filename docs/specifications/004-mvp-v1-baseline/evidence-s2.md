# Evidence — MVP Slice S2: install, first run, and Project setup

## Claim and authority boundary

This record covers the explicitly authorized Specification 004 Slice S2 only:
T04–T07. Work started from `main` revision
`66c0782ffbad2263f6a095e1c2c5e88100872a21`, the merge of PR #83. S3–S7,
GitHub Release publication, automatic upgrade/recovery, Provider mutation, and
final MVP acceptance were not authorized and were not performed.

The results below establish technical implementation Evidence. They do not imply
human acceptance. Issue or PR state also does not supply human acceptance.

## Delivered behavior

### T04 — release archive and owned install

- `scripts/build-release-archives.sh` accepts an exact semantic version, requires
  clean source for release provenance, refuses a non-empty output directory, and
  builds the three approved archive names:
  `macos-27-arm64`, `ubuntu-26.04-amd64`, and `ubuntu-26.04-arm64`.
- Every archive includes `lingo`, `LICENSE`, installer, closed release metadata,
  the compatible five-skill manifest and files, and a complete inner SHA-256
  manifest. `SHA256SUMS` identifies the outer archives.
- `scripts/install-release.sh` stages and rechecks the selected archive before
  extraction, rejects unlisted files and unsafe archive types/paths, validates
  host platform, skill and release schemas, and publishes only to explicit
  absolute user-owned destinations.
- The closed receipt binds destination, binary/archive/skill-manifest digests,
  version, revision, source state, platform, architecture, and skill-set version.
  Equivalent reinstall is `unchanged`; foreign, modified, symlinked, hard-linked,
  divergent, wrong-platform, or concurrent state is preserved and refused.
- The installer never edits a shell profile or `PATH`. S2 refuses owned version
  upgrade because resumable upgrade belongs to T20.

### T05 — five-skill compatibility and first run

- One versioned manifest closes the five hyphen-case embedded skills and records
  each SHA-256 digest plus binary compatibility version.
- `first-run` and `runtime codex status` report canonical completion/provenance
  plus every skill's expected digest and observed state: `missing`, `equivalent`,
  `owned_older`, or `modified_or_foreign`.
- Installation is idempotent, upgrades only known older Axiom bytes, refuses
  changed/foreign/link-unsafe files, retains a partial installed set after a
  deterministic interruption, and safely completes it on an explicit retry.
- First run directs `runtime codex install` and then explicit `project configure`;
  it does not invoke Codex, configure a Project, inspect Git, or edit a profile.

Current skill digests:

| Skill | SHA-256 |
|---|---|
| `axiom-project-configure` | `d481dc61ecd7a9afd1ffd0a79908515f15f04a75002a7d501eea5517f1f4844e` |
| `axiom-project-show` | `a80b3038c497fe3f3b817e5d5d28bca68de96d9ceaf76630f08ad1e34c5db0f4` |
| `axiom-work-item-create` | `6750abfe6cb817ff4f011d7c6b59a27e4b12832a7c1bbae68a9357bee249d4bd` |
| `axiom-work-item-run` | `a75f21684d38f325840461fbe8e959ed9fd2b925ac630c7d471147fdfef124dd` |
| `axiom-work-item-status` | `4fbb6fda699dc50af88f96355cb9cbed05dbebf34a7ed3218bc26b72b7fd60c7` |

### T06 — read-only normalized setup preview

- Guided and complete flags enter the same `projectapp.PrepareSetup` path and
  normalize Repository order before producing portable intent, local bindings,
  capability readiness, destinations, observed revisions, effects, and one
  preview digest.
- Preview performs no publication. Repository paths and their observations remain
  local; they do not enter `axiom.yaml`. No CWD or Git remote becomes identity or
  capability. Missing and unsupported Work Item capability remain explicit and
  block that journey only.
- The guided path asks only missing values, displays the same JSON proposal, and
  publishes only after the exact `yes` confirmation.

### T07 — authorized separate publication

- Apply requires the preview's explicit Project ID and digest plus
  `--authorize-local`, then re-observes portable state, local state, and every
  Repository directory. Changed facts yield canonical `denied_authority`.
- Portable intent publishes before ID-addressed local bindings. A confirmed
  portable commit followed by local failure reports canonical `partial` with the
  portable reference; it never claims rollback or atomicity across roots.
- Exact replay is a no-op. Two independent processes using one absent-state
  preview produce one winner and one fail-closed refusal (`denied_authority` or
  an in-progress-state inspection failure), never mixed success.
- The resulting Project resolves by UUID or slug from an unrelated CWD. Work Item
  use verifies the declared portable GitHub capability instead of silently
  falling back to ambient configuration.

## Deterministic fault and conflict observations

| Stage / condition | Test seam or case | Observed truth |
|---|---|---|
| F0–F3 before binary commit | injected `before_binary`, checksum/manifest/platform/link/type conflicts | no binary or receipt; owned marker/stage removed |
| F3 stale setup authority | changed inputs, Repository replacement, changed revisions | `denied_authority`; no Project publication |
| F4–F5 coordination/publication | existing lock, foreign destination, hard link, two-process Project race | conflict preserved; exactly one Project winner, loser refuses |
| F6 binary committed before receipt | injected `after_binary` | binary remains, receipt absent, marker says `binary_committed`, retry is `recovery_required` |
| F7 portable committed before local state | deterministic local-root replacement hook | canonical `partial`; portable digest remains readable |
| F8 cleanup | pre-commit marker cleanup plus S1 shared-publication F8 suite | commit truth preserved; unknown post-commit marker is not deleted automatically |

Timing sleeps are not race or fault oracles. Process coordination uses the shared
protected publication protocol and explicit lock/barrier state.

## Validation environment and commands

Native S2 execution environment:

- macOS 27.0, build `26A428`;
- Darwin 27.0.0, arm64, kernel `RELEASE_ARM64_T8103`;
- Go `go1.26.1 darwin/arm64`;
- filesystem tests use isolated temporary roots on the current host.

Commands used for the final branch validation are listed below. Each must exit
zero unless the command itself is a negative-path assertion inside a test:

```bash
bash -n scripts/build-release-archives.sh scripts/install-release.sh \
  scripts/test-release-archives.sh scripts/dogfood-poc.sh
go test ./... -count=1
go test -race ./internal/codexruntime ./internal/projectapp ./internal/local ./internal/cli ./cmd/lingo -count=1
go vet ./...
go build ./...
go mod verify
./scripts/test-release-archives.sh
./scripts/test-codex-skills.sh
./scripts/dogfood-poc.sh
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
git diff --check
```

Clean-source identified-artifact observation from implementation commit
`3a9e765dd8bfd6b5c1b9dc866ca1d960f3e440a9`, version `0.1.0-s2.1`:

| Archive | SHA-256 |
|---|---|
| `axiom-0.1.0-s2.1-macos-27-arm64.tar.gz` | `b11a17081fc82c79f9ece07894aff6da425d6f0fabb64f46c6c4bfa8eb98d6bd` |
| `axiom-0.1.0-s2.1-ubuntu-26.04-amd64.tar.gz` | `df66ea374f95c0f5988b140d5ada6437ae8c3f8e8f0bc74f9ec2ac513db42435` |
| `axiom-0.1.0-s2.1-ubuntu-26.04-arm64.tar.gz` | `64e2ca506c8ead1d7c3a2c3d14d3ba375748d2e3dbd175ee7953478e249c4c98` |

The native macOS archive installed with `install_status=installed`; its binary
reported version `0.1.0-s2.1`, revision `3a9e765dd8bf`, and source state `clean`.
The installed binary digest was
`51d55c8e358196110a3173cc2e0ffa5072af8bf988e78d87a64ee5256fca5566`;
the closed receipt digest was
`cc276213c36d72001747d9a60d5c6fe8fbedcf31383f73011a1a604e76f6f0a1`.
These artifacts were generated only in an isolated temporary directory for
Evidence; they were not published as a prerelease or GitHub Release.

## Native Evidence and limitations

- The macOS 27/arm64 archive is built and installed natively in this Evidence run.
- Ubuntu 26.04 amd64 and arm64 binaries/archives are cross-compiled and their
  structure/manifests/checksums are verified here. This is not substituted for
  native ext4 execution.
- Exact Ubuntu point-release/kernel/ext4 runs, ACL matrices, all target-specific
  primitives, owned upgrade/resume, and the complete target matrix remain explicit
  blocking obligations of T20, T22, and T24.
- Distribution trust in S2 is SHA-256 integrity only. Signing, notarization,
  package managers, automatic update, and GitHub Release publication remain out
  of scope.
- Tests cover supported local-process concurrency and deterministic injected
  faults, not physical power loss or malicious arbitrary same-UID interleavings.

## Acceptance status

S2 T04–T07 are technically implemented and ready for human review. Human
acceptance is not claimed. No authority exists here to start T08 or any S3–S7
Task.
